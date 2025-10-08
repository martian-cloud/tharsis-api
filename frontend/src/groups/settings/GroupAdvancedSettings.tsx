import { useState } from 'react';
import {
    Alert,
    AlertTitle,
    Box,
    Button,
    Collapse,
    Dialog,
    DialogActions,
    DialogTitle,
    DialogContent,
    TextField,
    Typography
} from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import LoadingButton from '@mui/lab/LoadingButton';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { atomDark as prismTheme } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { useFragment, useMutation } from 'react-relay';
import { useSnackbar } from 'notistack';
import { useNavigate } from 'react-router-dom';
import SettingsToggleButton from '../../common/SettingsToggleButton';
import MigrateGroupDialog from './MigrateGroupDialog';
import { GroupAdvancedSettingsFragment_group$key } from './__generated__/GroupAdvancedSettingsFragment_group.graphql';
import { GroupAdvancedSettingsDeleteDialogFragment_group$key } from './__generated__/GroupAdvancedSettingsDeleteDialogFragment_group.graphql';
import { GroupAdvancedSettingsDeleteMutation } from './__generated__/GroupAdvancedSettingsDeleteMutation.graphql';

interface ConfirmationDialogProps {
    deleteInProgress: boolean
    onClose: (confirm?: boolean) => void
    closeDialog: () => void
    open: boolean
    fragmentRef: GroupAdvancedSettingsDeleteDialogFragment_group$key
}

interface Props {
    fragmentRef: GroupAdvancedSettingsFragment_group$key
}

function DeleteConfirmationDialog(props: ConfirmationDialogProps) {
    const { deleteInProgress, onClose, closeDialog, open, fragmentRef } = props;
    const [deleteInput, setDeleteInput] = useState<string>('');

    const data = useFragment(
        graphql`
        fragment GroupAdvancedSettingsDeleteDialogFragment_group on Group
        {
            name
            fullPath
        }
    `, fragmentRef);

    return (
        <Dialog
            keepMounted
            maxWidth="sm"
            open={open}
        >
            <DialogTitle>Delete Group</DialogTitle>
            <DialogContent >
                <Alert sx={{ mb: 2 }} severity="warning">
                    <AlertTitle>Warning</AlertTitle>
                    Deleting a group is an <strong><ins>irreversible</ins></strong> operation. All nested groups and/or workspaces with their associated deployment states will be deleted and <strong><ins>cannot be recovered</ins></strong>.
                </Alert>
                <Typography variant="subtitle2">Enter the following to confirm deletion:</Typography>
                <SyntaxHighlighter style={prismTheme} customStyle={{ fontSize: 14, marginBottom: 14 }} children={data.fullPath} />
                <TextField
                    autoComplete="off"
                    fullWidth
                    size="small"
                    placeholder={data.fullPath}
                    value={deleteInput}
                    onChange={(event: any) => setDeleteInput(event.target.value)}
                    >
                </TextField>
            </DialogContent>
            <DialogActions>
                <Button
                    color="inherit"
                    onClick={() => {
                        closeDialog()
                        setDeleteInput('')
                    }}>Cancel</Button>
                <LoadingButton
                    color="error"
                    variant="outlined"
                    loading={deleteInProgress}
                    disabled={data.fullPath !== deleteInput}
                    onClick={() => {
                        onClose(true)
                        setDeleteInput('')
                    }}>Delete</LoadingButton>
            </DialogActions>
        </Dialog>
    );
}

function GroupAdvancedSettings({ fragmentRef }: Props) {
    const [showDeleteConfirmationDialog, setShowDeleteConfirmationDialog] = useState<boolean>(false);
    const [showMigrateGroupDialog, setShowMigrateGroupDialog] = useState<boolean>(false);
    const [showSettings, setShowSettings] = useState<boolean>(false);
    const { enqueueSnackbar } = useSnackbar();
    const navigate = useNavigate();

    const data = useFragment(
        graphql`
        fragment GroupAdvancedSettingsFragment_group on Group
        {
            name
            fullPath
            ...GroupAdvancedSettingsDeleteDialogFragment_group
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

                    if (deleteData.deleteGroup.problems.length) {
                        enqueueSnackbar(deleteData.deleteGroup.problems.map(problem => problem.message).join('; '), { variant: 'warning' });
                    } else navigate(`../${data.fullPath.slice(0, -data.name.length - 1)}`);
                },
                onError: error => {
                    setShowDeleteConfirmationDialog(false);
                    enqueueSnackbar(`An unexpected error occurred: ${error.message}`, { variant: 'error' });
                }
            });
        } else {
            setShowDeleteConfirmationDialog(false)
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
                <DeleteConfirmationDialog
                    fragmentRef={data}
                    deleteInProgress={commitDeleteInFlight}
                    onClose={onDeleteConfirmationDialogClosed}
                    closeDialog={() => setShowDeleteConfirmationDialog(false)}
                    open={showDeleteConfirmationDialog}
                />
            </Collapse>
        </Box>
    );
}

export default GroupAdvancedSettings
