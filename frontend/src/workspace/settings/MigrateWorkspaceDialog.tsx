import { LoadingButton } from '@mui/lab';
import { Alert, AlertTitle, Button, Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import { useCallback, useState } from 'react';
import { useFragment, useMutation } from 'react-relay';
import { useNavigate } from 'react-router-dom';
import { MutationError } from '../../common/error';
import GroupAutocomplete, { GroupOption } from '../../groups/GroupAutocomplete';
import { MigrateWorkspaceDialogFragment_workspace$key } from './__generated__/MigrateWorkspaceDialogFragment_workspace.graphql';
import { MigrateWorkspaceDialogMutation } from './__generated__/MigrateWorkspaceDialogMutation.graphql';

interface Props {
    onClose: () => void
    fragmentRef: MigrateWorkspaceDialogFragment_workspace$key
}

function MigrateWorkspaceDialog({ onClose, fragmentRef }: Props) {
    const navigate = useNavigate();
    const { enqueueSnackbar } = useSnackbar();
    const [newGroupPath, setNewGroupPath] = useState<string>('');
    const [error, setError] = useState<MutationError>();

    const workspace = useFragment<MigrateWorkspaceDialogFragment_workspace$key>(
        graphql`
        fragment MigrateWorkspaceDialogFragment_workspace on Workspace
        {
            name
            fullPath
            groupPath
        }
    `, fragmentRef);

    const [commit, isInFlight] = useMutation<MigrateWorkspaceDialogMutation>(graphql`
        mutation MigrateWorkspaceDialogMutation($input: MigrateWorkspaceInput!) {
            migrateWorkspace(input: $input) {
                workspace {
                    id
                    fullPath
                    groupPath
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const filterGroups = useCallback((options: readonly GroupOption[]) => {
        return options.filter((opt: GroupOption) => (opt.fullPath !== workspace.groupPath));
    }, [workspace]);

    const onMigrate = () => {
        commit({
            variables: {
                input: {
                    workspacePath: workspace.fullPath,
                    newGroupPath: newGroupPath
                }
            },
            onCompleted: data => {
                if (data.migrateWorkspace.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.migrateWorkspace.problems.map(problem => problem.message).join('; ')
                    });
                } else if (!data.migrateWorkspace.workspace) {
                    setError({
                        severity: 'error',
                        message: "Unexpected error occurred"
                    });
                } else {
                    onClose()
                    navigate(`../${data.migrateWorkspace.workspace.fullPath}`)
                    enqueueSnackbar(`${workspace.name} has been migrated to ${data.migrateWorkspace.workspace.fullPath}`, { variant: 'success' });
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
        setNewGroupPath(group?.fullPath);
    };

    return (
        <Dialog
            keepMounted
            maxWidth="sm"
            open
        >
            <DialogTitle>Migrate Workspace</DialogTitle>
            <DialogContent dividers>
                {error && <Alert sx={{ mb: 2 }} severity={error.severity}>
                    {error.message}
                </Alert>}
                <Alert sx={{ mb: 2 }} severity="warning">
                    <AlertTitle>Warning</AlertTitle>
                    Resources <i>within</i> the workspace are automatically migrated to the new workspace path. However, <strong><ins>any inherited resource assignments</ins></strong>, such as, managed identities, service account memberships, VCS Provider links etc., that are not available in the new workspace hierarchy, <strong><ins>will automatically be removed</ins></strong>.
                </Alert>
                <GroupAutocomplete
                    placeholder="Select a parent group"
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
                    disabled={!newGroupPath}
                    color="primary"
                    variant="outlined"
                    loading={isInFlight}
                    onClick={onMigrate}>Migrate
                </LoadingButton>
            </DialogActions>
        </Dialog>
    );
}

export default MigrateWorkspaceDialog;
