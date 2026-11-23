import { useState } from 'react'
import {
    Alert,
    AlertTitle,
    Box,
    Button,
    Typography
} from '@mui/material'
import { useFragment, useMutation } from 'react-relay';
import { useSnackbar } from 'notistack';
import { useNavigate } from 'react-router-dom';
import graphql from 'babel-plugin-relay/macro';
import ConfirmationDialog from '../../common/ConfirmationDialog';
import SettingsSection from '../../common/SettingsSection';
import MigrateWorkspaceDialog from './MigrateWorkspaceDialog';
import { WorkspaceAdvancedSettingsFragment_workspace$key } from './__generated__/WorkspaceAdvancedSettingsFragment_workspace.graphql'
import { WorkspaceAdvancedSettingsDeleteMutation } from './__generated__/WorkspaceAdvancedSettingsDeleteMutation.graphql'

interface Props {
    fragmentRef: WorkspaceAdvancedSettingsFragment_workspace$key
}

function WorkspaceAdvancedSettings({ fragmentRef }: Props) {
    const [showDeleteConfirmationDialog, setShowDeleteConfirmationDialog] = useState<boolean>(false);
    const [showMigrateWorkspaceDialog, setShowMigrateWorkspaceDialog] = useState<boolean>(false);
    const { enqueueSnackbar } = useSnackbar();
    const navigate = useNavigate();

    const data = useFragment(
        graphql`
        fragment WorkspaceAdvancedSettingsFragment_workspace on Workspace
        {
            name
            fullPath
            ...MigrateWorkspaceDialogFragment_workspace
        }
    `, fragmentRef
    )

    const [commitDelete, commitDeleteInFlight] = useMutation<WorkspaceAdvancedSettingsDeleteMutation>(
        graphql`
        mutation WorkspaceAdvancedSettingsDeleteMutation($input: DeleteWorkspaceInput!) {
            deleteWorkspace(input: $input){
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const onDeleteConfirmationDialogClosed = (confirm?: boolean) => {
        if (confirm) {
            commitDelete({
                variables: {
                    input: {
                        workspacePath: data.fullPath,
                        force: true
                    },
                },
                // Namespace lists are cached per view, search and expanded group, so mark the whole store stale and let
                // whichever list mounts next refetch instead of patching each connection
                updater: (store, data) => {
                    if (!data?.deleteWorkspace.problems.length) {
                        store.invalidateStore();
                    }
                },
                onCompleted: deleteData => {
                    setShowDeleteConfirmationDialog(false);
                    if (deleteData.deleteWorkspace.problems.length) {
                        enqueueSnackbar(deleteData.deleteWorkspace.problems.map(problem => problem.message).join('; '), { variant: 'warning' });
                    } else navigate(`../${data.fullPath.slice(0, -data.name.length - 1)}`);
                },
                onError: error => {
                    setShowDeleteConfirmationDialog(false);
                    enqueueSnackbar(`An unexpected error occurred: ${error.message}`, { variant: 'error' });
                }
            });
        } else {
            setShowDeleteConfirmationDialog(false);
        }
    };

    return (
        <Box>
            <SettingsSection section="advanced" title="Advanced Settings">
                <Box>
                    <Box sx={{ mb: 4 }}>
                        <Typography variant="subtitle1" gutterBottom>Migrate Workspace</Typography>
                        <Typography variant="subtitle2" sx={{
                            marginBottom: 2
                        }}>Migrate workspace to another group</Typography>
                        <Alert sx={{ mb: 2 }} severity="warning">Migrating a workspace may result in changes to its configuration if any assigned resources are not available in the new group.</Alert>
                        <Button
                            variant="outlined"
                            color="warning"
                            onClick={() => setShowMigrateWorkspaceDialog(true)}
                        >Migrate Workspace</Button>
                    </Box>
                    <Typography variant="subtitle1" gutterBottom>Delete Workspace</Typography>
                    <Alert sx={{ mb: 2 }} severity="error">Deleting a workspace is a permanent action that cannot be undone.</Alert>
                    <Box>
                        <Button variant="outlined" color="error" onClick={() => setShowDeleteConfirmationDialog(true)}>Delete Workspace</Button>
                    </Box>
                </Box>
            </SettingsSection>
            {showMigrateWorkspaceDialog && <MigrateWorkspaceDialog onClose={() => setShowMigrateWorkspaceDialog(false)} fragmentRef={data} />}
            {showDeleteConfirmationDialog && (
                <ConfirmationDialog
                    title="Delete Workspace"
                    maxWidth="sm"
                    confirmLabel="Delete"
                    typedConfirmation={{ label: 'Enter the following to confirm deletion:', value: data.fullPath }}
                    confirmInProgress={commitDeleteInFlight}
                    onConfirm={() => onDeleteConfirmationDialogClosed(true)}
                    onClose={() => onDeleteConfirmationDialogClosed()}
                >
                    <Alert sx={{ mb: 2 }} severity="warning">
                        <AlertTitle>Warning</AlertTitle>
                        Deleting a workspace is an <strong><ins>irreversible</ins></strong> operation. All state files, resources, and data associated with this workspace will be deleted and <strong><ins>cannot be recovered</ins></strong>.
                    </Alert>
                </ConfirmationDialog>
            )}
        </Box>
    );
}

export default WorkspaceAdvancedSettings;
