import { LoadingButton } from '@mui/lab';
import { Alert, AlertTitle, Button, Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import { useCallback, useMemo, useState } from 'react';
import { useFragment, useMutation } from 'react-relay';
import { useNavigate } from 'react-router-dom';
import { MutationError } from '../../common/error';
import GroupAutocomplete, { GroupOption } from '../GroupAutocomplete';
import { MigrateGroupDialogFragment_group$key } from './__generated__/MigrateGroupDialogFragment_group.graphql';
import { MigrateGroupDialogMutation } from './__generated__/MigrateGroupDialogMutation.graphql';

interface Props {
    onClose: () => void
    fragmentRef: MigrateGroupDialogFragment_group$key
}

function MigrateGroupDialog({ onClose, fragmentRef }: Props) {
    const navigate = useNavigate();
    const { enqueueSnackbar } = useSnackbar();
    const [newParentPath, setNewParentPath] = useState<string>('');
    const [error, setError] = useState<MutationError>();

    const group = useFragment<MigrateGroupDialogFragment_group$key>(
        graphql`
        fragment MigrateGroupDialogFragment_group on Group
        {
            name
            fullPath
        }
    `, fragmentRef);

    const [commit, isInFlight] = useMutation<MigrateGroupDialogMutation>(graphql`
        mutation MigrateGroupDialogMutation($input: MigrateGroupInput!) {
            migrateGroup(input: $input) {
                group {
                    id
                    fullPath
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const parentPath = useMemo(() => {
        const index = group.fullPath.lastIndexOf('/');
        return index !== -1 ? group.fullPath.substring(0, index) : '';
    }, [group]);

    const filterGroups = useCallback((options: readonly GroupOption[]) => {
        return options.filter((opt: GroupOption) => (!opt.fullPath.startsWith(`${group.fullPath}/`) && opt.fullPath !== group.fullPath && opt.fullPath !== parentPath));
    }, [group, parentPath]);

    const onMigrate = () => {
        commit({
            variables: {
                input: {
                    groupPath: group.fullPath,
                    newParentPath: newParentPath === '<< no parent >>' ? null : newParentPath
                }
            },
            onCompleted: data => {
                if (data.migrateGroup.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.migrateGroup.problems.map(problem => problem.message).join('; ')
                    });
                } else if (!data.migrateGroup.group) {
                    setError({
                        severity: 'error',
                        message: "Unexpected error occurred"
                    });
                } else {
                    onClose()
                    navigate(`../${data.migrateGroup.group.fullPath}`)
                    enqueueSnackbar(`${group.name} has been migrated to ${newParentPath === '<< no parent >>' ? 'the root level.' : data.migrateGroup.group.fullPath}`, { variant: 'success' });
                }
            },
            onError: error => {
                setError({
                    severity: 'error',
                    message: `Unexpected error occurred: ${error.message}`
                });
            }
        })
    };

    const onGroupChange = (group: any) => {
        setNewParentPath(group?.fullPath)
    };

    return (
        <Dialog
            keepMounted
            maxWidth="sm"
            open
        >
            <DialogTitle>Migrate Group</DialogTitle>
            <DialogContent dividers>
                {error && <Alert sx={{ mb: 2 }} severity={error.severity}>
                    {error.message}
                </Alert>}
                <Alert sx={{ mb: 2 }} severity="warning">
                    <AlertTitle>Warning</AlertTitle>
                    Resources <i>within</i> the group, as well as child groups and their resources, are automatically migrated to the new group path. However, <strong><ins>any inherited resource assignments</ins></strong>, such as, managed identities, service accounts, etc, that are not available in the new group hierarchy, <strong><ins>will automatically be removed</ins></strong>. If the group is migrated to a sibling group or within the current group's hierarchy, inherited resource assignments will stay in place.
                </Alert>
                <GroupAutocomplete
                    placeholder="Select a parent group"
                    includeNoParentOption={parentPath !== ''}
                    sx={{ mb: 2 }}
                    onSelected={onGroupChange}
                    filterGroups={filterGroups}
                />
            </DialogContent>
            <DialogActions>
                <Button
                    color="inherit"
                    onClick={onClose}>Cancel
                </Button>
                <LoadingButton
                    disabled={!newParentPath}
                    color="primary"
                    variant="outlined"
                    loading={isInFlight}
                    onClick={onMigrate}>Migrate
                </LoadingButton>
            </DialogActions>
        </Dialog>
    );
}

export default MigrateGroupDialog
