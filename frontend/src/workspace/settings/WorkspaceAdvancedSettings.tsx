import { useState } from 'react'
import {
    Alert,
    AlertTitle,
    Box,
    Button,
    Collapse,
    TextField,
    Typography
} from '@mui/material'
import { GetConnections } from '../../groups/WorkspaceList'
import { useFragment, useMutation } from 'react-relay';
import { useSnackbar } from 'notistack';
import { useNavigate } from 'react-router-dom';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { atomDark as prismTheme } from 'react-syntax-highlighter/dist/esm/styles/prism';
import graphql from 'babel-plugin-relay/macro';
import ConfirmationDialog from '../../common/ConfirmationDialog';
import SettingsToggleButton from '../../common/SettingsToggleButton';
import MigrateWorkspaceDialog from './MigrateWorkspaceDialog';
import { WorkspaceAdvancedSettingsFragment_workspace$key } from './__generated__/WorkspaceAdvancedSettingsFragment_workspace.graphql'
import { WorkspaceAdvancedSettingsDeleteMutation } from './__generated__/WorkspaceAdvancedSettingsDeleteMutation.graphql'

interface Props {
    fragmentRef: WorkspaceAdvancedSettingsFragment_workspace$key
}

function WorkspaceAdvancedSettings({ fragmentRef }: Props) {
    const [showDeleteConfirmationDialog, setShowDeleteConfirmationDialog] = useState<boolean>(false);
    const [showMigrateWorkspaceDialog, setShowMigrateWorkspaceDialog] = useState<boolean>(false);
    const [showSettings, setShowSettings] = useState<boolean>(false);
    const [confirmInput, setConfirmInput] = useState('');
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
        mutation WorkspaceAdvancedSettingsDeleteMutation($input: DeleteWorkspaceInput!, $connections: [ID!]!) {
            deleteWorkspace(input: $input){
                workspace {
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

    const onDeleteConfirmationDialogClosed = (confirm?: boolean) => {
        if (confirm) {
            commitDelete({
                variables: {
                    input: {
                        workspacePath: data.fullPath,
                        force: true
                    },
                    connections: GetConnections(data.fullPath.substring(0, ((data.fullPath.length - data.name.length - 1))))
                },
                onCompleted: deleteData => {
                    setShowDeleteConfirmationDialog(false);
                    setConfirmInput('');
                    if (deleteData.deleteWorkspace.problems.length) {
                        enqueueSnackbar(deleteData.deleteWorkspace.problems.map(problem => problem.message).join('; '), { variant: 'warning' });
                    } else navigate(`../${data.fullPath.slice(0, -data.name.length - 1)}`);
                },
                onError: error => {
                    setShowDeleteConfirmationDialog(false);
                    setConfirmInput('');
                    enqueueSnackbar(`An unexpected error occurred: ${error.message}`, { variant: 'error' });
                }
            });
        } else {
            setShowDeleteConfirmationDialog(false);
            setConfirmInput('');
        }
    };

    return (
        <Box>
            <SettingsToggleButton
                title="Advanced Settings"
                showSettings={showSettings}
                onToggle={() => setShowSettings(!showSettings)}
            />
            <Collapse
                in={showSettings}
                timeout="auto"
                unmountOnExit
            >
                <Box>
                    <Box sx={{ mb: 4 }}>
                        <Typography variant="subtitle1" gutterBottom>Migrate Workspace</Typography>
                        <Typography marginBottom={2} variant="subtitle2">Migrate workspace to another group</Typography>
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
            </Collapse>
            {showMigrateWorkspaceDialog && <MigrateWorkspaceDialog onClose={() => setShowMigrateWorkspaceDialog(false)} fragmentRef={data} />}
            {showDeleteConfirmationDialog && (
                <ConfirmationDialog
                    title="Delete Workspace"
                    maxWidth="sm"
                    confirmLabel="Delete"
                    confirmDisabled={data.fullPath !== confirmInput}
                    confirmInProgress={commitDeleteInFlight}
                    onConfirm={() => onDeleteConfirmationDialogClosed(true)}
                    onClose={() => onDeleteConfirmationDialogClosed()}
                >
                    <Alert sx={{ mb: 2 }} severity="warning">
                        <AlertTitle>Warning</AlertTitle>
                        Deleting a workspace is an <strong><ins>irreversible</ins></strong> operation. All state files, resources, and data associated with this workspace will be deleted and <strong><ins>cannot be recovered</ins></strong>.
                    </Alert>
                    <Typography variant="subtitle2">Enter the following to confirm deletion:</Typography>
                    <SyntaxHighlighter style={prismTheme} customStyle={{ fontSize: 14, marginBottom: 14 }}>{data.fullPath}</SyntaxHighlighter>
                    <TextField
                        autoComplete="off"
                        fullWidth
                        size="small"
                        placeholder={data.fullPath}
                        value={confirmInput}
                        onChange={(e) => setConfirmInput(e.target.value)}
                    />
                </ConfirmationDialog>
            )}
        </Box>
    );
}

export default WorkspaceAdvancedSettings;
