import { useState } from 'react';
import {
    Alert,
    AlertTitle,
    Box,
    Button,
    Collapse,
    TextField,
    Typography
} from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { atomDark as prismTheme } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { useFragment, useMutation } from 'react-relay';
import { useSnackbar } from 'notistack';
import { useNavigate } from 'react-router-dom';
import ConfirmationDialog from '../../common/ConfirmationDialog';
import SettingsToggleButton from '../../common/SettingsToggleButton';
import MigrateGroupDialog from './MigrateGroupDialog';
import { GroupAdvancedSettingsFragment_group$key } from './__generated__/GroupAdvancedSettingsFragment_group.graphql';
import { GroupAdvancedSettingsDeleteMutation } from './__generated__/GroupAdvancedSettingsDeleteMutation.graphql';

interface Props {
    fragmentRef: GroupAdvancedSettingsFragment_group$key
}

function GroupAdvancedSettings({ fragmentRef }: Props) {
    const [showDeleteConfirmationDialog, setShowDeleteConfirmationDialog] = useState<boolean>(false);
    const [showMigrateGroupDialog, setShowMigrateGroupDialog] = useState<boolean>(false);
    const [showSettings, setShowSettings] = useState<boolean>(false);
    const [confirmInput, setConfirmInput] = useState('');
    const { enqueueSnackbar } = useSnackbar();
    const navigate = useNavigate();

    const data = useFragment(
        graphql`
        fragment GroupAdvancedSettingsFragment_group on Group
        {
            name
            fullPath
            ...MigrateGroupDialogFragment_group
        }
    `, fragmentRef);

    const [commitDelete, commitDeleteInFlight] = useMutation<GroupAdvancedSettingsDeleteMutation>(
        graphql`
        mutation GroupAdvancedSettingsDeleteMutation($input: DeleteGroupInput! ) {
            deleteGroup(input: $input){
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
                        groupPath: data.fullPath,
                        force: true
                    }
                },
                onCompleted: deleteData => {
                    setShowDeleteConfirmationDialog(false);
                    setConfirmInput('');
                    if (deleteData.deleteGroup.problems.length) {
                        enqueueSnackbar(deleteData.deleteGroup.problems.map(problem => problem.message).join('; '), { variant: 'warning' });
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
                <Box sx={{ mb: 4 }}>
                    <Typography variant="subtitle1" gutterBottom>Migrate Group</Typography>
                    <Typography marginBottom={2} variant="subtitle2">Migrate group to another parent or sibling group</Typography>
                    <Alert sx={{ mb: 2 }} severity="warning">Migrating a group is potentially destructive.</Alert>
                    <Button
                        variant="outlined"
                        color="warning"
                        onClick={() => setShowMigrateGroupDialog(true)}
                    >Migrate Group</Button>
                </Box>
                <Typography variant="subtitle1" gutterBottom>Delete Group</Typography>
                <Alert sx={{ mb: 2 }} severity="error">Deleting a group is a permanent action that cannot be undone.</Alert>
                <Box>
                    <Button
                        variant="outlined"
                        color="error"
                        onClick={() => setShowDeleteConfirmationDialog(true)}
                    >Delete Group</Button>
                </Box>
                {showMigrateGroupDialog && <MigrateGroupDialog onClose={() => setShowMigrateGroupDialog(false)} fragmentRef={data} />}
                {showDeleteConfirmationDialog && (
                    <ConfirmationDialog
                        title="Delete Group"
                        maxWidth="sm"
                        confirmLabel="Delete"
                        confirmDisabled={data.fullPath !== confirmInput}
                        confirmInProgress={commitDeleteInFlight}
                        onConfirm={() => onDeleteConfirmationDialogClosed(true)}
                        onClose={() => onDeleteConfirmationDialogClosed()}
                    >
                        <Alert sx={{ mb: 2 }} severity="warning">
                            <AlertTitle>Warning</AlertTitle>
                            Deleting a group is an <strong><ins>irreversible</ins></strong> operation. All nested groups and/or workspaces with their associated deployment states will be deleted and <strong><ins>cannot be recovered</ins></strong>.
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
            </Collapse>
        </Box>
    );
}

export default GroupAdvancedSettings
