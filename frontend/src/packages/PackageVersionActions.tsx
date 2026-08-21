import ArrowDropDownIcon from '@mui/icons-material/ArrowDropDown';
import { Button, ButtonGroup, Menu, MenuItem } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import { useState } from 'react';
import { useMutation } from 'react-relay/hooks';
import { Link as RouterLink, useNavigate } from 'react-router-dom';
import { ConnectionHandler } from 'relay-runtime';
import ConfirmationDialog from '../common/ConfirmationDialog';
import { PackageVersionActionsDeleteMutation } from './__generated__/PackageVersionActionsDeleteMutation.graphql';

// VersionActionTarget is the version the options act on: the one currently on screen, rather than a row
// picked out of the version history.
export interface VersionActionTarget {
    id: string
    version: string
    status: string
}

interface Props {
    packageId: string
    groupPath: string
    // Editing a version rewrites the files of an already-published version, which the api only permits
    // when the package opts into mutable versions. With it off, versions are immutable once uploaded and
    // the option isn't offered at all.
    allowMutableVersions: boolean
    // Null until something has been published, which leaves creating a version as the only option.
    version: VersionActionTarget | null
    // Deleting the version on screen means the page has to let go of it, and deleting the latest promotes
    // another one, so the caller re-reads the package.
    onVersionDeleted: (versionId: string) => void
}

// PackageVersionActions is the group page's header control for a package's versions: creating one is the
// primary action, with editing and deleting the version being viewed offered under it. They live here
// rather than on each row of the history because they act on one version at a time, and the row actions
// crowded a list whose job is choosing which version to look at.
function PackageVersionActions({ packageId, groupPath, allowMutableVersions, version, onVersionDeleted }: Props) {
    const navigate = useNavigate();
    const { enqueueSnackbar } = useSnackbar();
    const [menuAnchorEl, setMenuAnchorEl] = useState<Element | null>(null);
    const [showDeleteConfirmation, setShowDeleteConfirmation] = useState(false);

    const [commitDelete, deleteInFlight] = useMutation<PackageVersionActionsDeleteMutation>(graphql`
        mutation PackageVersionActionsDeleteMutation($input: DeletePackageVersionInput!, $connections: [ID!]!) {
            deletePackageVersion(input: $input) {
                packageVersion {
                    id @deleteEdge(connections: $connections)
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const onMenuAction = (action: () => void) => {
        setMenuAnchorEl(null);
        action();
    };

    // Both options act on a published version, so until there is one the menu would be empty and the
    // split button collapses to the plain create button. Derived from the options themselves rather than
    // from the version alone, so adding or gating an option keeps the arrow honest.
    const canEdit = version != null && allowMutableVersions;
    const canDelete = version != null;
    const hasOptions = canEdit || canDelete;

    const onDelete = (confirm?: boolean) => {
        if (!confirm || !version) {
            setShowDeleteConfirmation(false);
            return;
        }

        const deletedId = version.id;

        commitDelete({
            variables: {
                input: { id: deletedId },
                // The connection is owned by the package record, so its id is derived from the package.
                // The history is only mounted on the Versions tab; with it closed there is no connection
                // to update and dropping the edge is simply a no-op.
                connections: [ConnectionHandler.getConnectionID(
                    packageId,
                    'PackageVersionList_versions',
                    { sort: 'CREATED_AT_DESC' }
                )],
            },
            onCompleted: response => {
                setShowDeleteConfirmation(false);
                if (response.deletePackageVersion.problems.length) {
                    enqueueSnackbar(
                        response.deletePackageVersion.problems.map(problem => problem.message).join('; '),
                        { variant: 'warning' }
                    );
                    return;
                }
                onVersionDeleted(deletedId);
            },
            onError: error => {
                setShowDeleteConfirmation(false);
                enqueueSnackbar(`Unexpected error occurred: ${error.message}`, { variant: 'error' });
            }
        });
    };

    return (
        <>
            <ButtonGroup variant="outlined" color="primary" size="small">
                <Button
                    component={RouterLink}
                    to={`/groups/${groupPath}/-/packages/${packageId}/versions/new`}
                >
                    Create new version
                </Button>
                {hasOptions && <Button
                    aria-label="package version options menu"
                    aria-haspopup="menu"
                    onClick={event => setMenuAnchorEl(event.currentTarget)}
                >
                    <ArrowDropDownIcon fontSize="small" />
                </Button>}
            </ButtonGroup>
            <Menu
                id="package-version-options-menu"
                anchorEl={menuAnchorEl}
                open={Boolean(menuAnchorEl)}
                onClose={() => setMenuAnchorEl(null)}
                anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
                transformOrigin={{ vertical: 'top', horizontal: 'right' }}
            >
                {/* Both options act on whichever version is being viewed, rather than on the row the
                    menu is nearest to. */}
                {canEdit && <MenuItem
                    // A version whose upload never finished has no files to edit.
                    disabled={version?.status !== 'UPLOADED'}
                    onClick={() => onMenuAction(() => navigate(
                        `/groups/${groupPath}/-/packages/${packageId}/versions/${version?.id}/edit`
                    ))}
                >
                    Edit version
                </MenuItem>}
                {canDelete && <MenuItem onClick={() => onMenuAction(() => setShowDeleteConfirmation(true))}>
                    Delete version
                </MenuItem>}
            </Menu>
            {/* Rendered outside the Menu so closing the menu doesn't unmount the dialog. */}
            {showDeleteConfirmation && version && <ConfirmationDialog
                title="Delete Package Version"
                confirmLabel="Delete"
                confirmInProgress={deleteInFlight}
                onConfirm={() => onDelete(true)}
                onClose={() => onDelete()}
            >
                Are you sure you want to delete version <strong>{version.version}</strong>?
            </ConfirmationDialog>}
        </>
    );
}

export default PackageVersionActions;
