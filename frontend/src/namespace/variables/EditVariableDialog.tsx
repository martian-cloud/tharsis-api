import { Alert, Checkbox, Chip, FormControlLabel, Typography } from '@mui/material';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import { useTheme } from '@mui/material/styles';
import TextField from '@mui/material/TextField';
import useMediaQuery from '@mui/material/useMediaQuery';
import graphql from 'babel-plugin-relay/macro';
import moment from 'moment';
import * as React from 'react';
import { useEffect, useMemo, useState } from 'react';
import { useMutation } from 'react-relay';
import { MutationError } from '../../common/error';
import { EditVariableDialogCreateVariableMutation } from './__generated__/EditVariableDialogCreateVariableMutation.graphql';
import { EditVariableDialogUpdateVariableMutation } from './__generated__/EditVariableDialogUpdateVariableMutation.graphql';
import LoadingButton from '@mui/lab/LoadingButton';

interface Props {
    variable: any;
    namespacePath: string;
    onClose: () => void;
}

function EditVariableDialog(props: Props) {
    const { variable, namespacePath, onClose } = props;
    const theme = useTheme();
    const fullScreen = useMediaQuery(theme.breakpoints.down('md'));

    const [error, setError] = useState<MutationError | null>();
    const [editedVariable, setEditedVariable] = useState<any>(null);

    const [commitUpdateVariable, updateInFlight] = useMutation<EditVariableDialogUpdateVariableMutation>(graphql`
        mutation EditVariableDialogUpdateVariableMutation($input: UpdateNamespaceVariableInput!) {
            updateNamespaceVariable(input: $input) {
                namespace {
                    id
                    variables {
                        ...VariableListItemFragment_variable
                    }
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const [commitCreateVariable, createInFlight] = useMutation<EditVariableDialogCreateVariableMutation>(graphql`
        mutation EditVariableDialogCreateVariableMutation($input: CreateNamespaceVariableInput!) {
            createNamespaceVariable(input: $input) {
                namespace {
                    id
                    variables {
                        ...VariableListItemFragment_variable
                    }
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    useEffect(() => {
        setEditedVariable({ ...variable, value: variable.value ?? '' });
        setError(null);
    }, [variable]);

    const saveVariable = () => {
        setError(null);
        if (editedVariable.id) {
            commitUpdateVariable({
                variables: {
                    input: {
                        id: editedVariable.id,
                        key: editedVariable.key,
                        value: editedVariable.value,
                    },
                },
                onCompleted: data => {
                    if (data.updateNamespaceVariable.problems.length) {
                        setError({
                            severity: 'warning',
                            message: data.updateNamespaceVariable.problems.map(problem => problem.message).join('; ')
                        });
                    } else {
                        onClose();
                    }
                },
                onError: error => {
                    setError({
                        severity: 'error',
                        message: `Unexpected Error Occurred: ${error.message}`
                    });
                }
            });
        } else {
            commitCreateVariable({
                variables: {
                    input: {
                        namespacePath: namespacePath,
                        category: editedVariable.category,
                        sensitive: editedVariable.sensitive,
                        key: editedVariable.key,
                        value: editedVariable.value,
                    },
                },
                onCompleted: data => {
                    if (data.createNamespaceVariable.problems.length) {
                        setError({
                            severity: 'warning',
                            message: data.createNamespaceVariable.problems.map(problem => problem.message).join('; ')
                        });
                    } else {
                        onClose();
                    }
                },
                onError: error => {
                    setError({
                        severity: 'error',
                        message: `Unexpected Error Occurred: ${error.message}`
                    });
                }
            });
        }
    }

    const onFieldChange = (event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
        setEditedVariable({ ...editedVariable, [event.target.name]: event.target.value });
    };

    const onSensitiveChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        setEditedVariable({ ...editedVariable, sensitive: event.target.checked });
    }

    const editMode = useMemo(() => !!editedVariable?.id, [editedVariable?.id]);

    return editedVariable ? (
        <Dialog
            fullWidth
            maxWidth="md"
            fullScreen={fullScreen}
            open={!!variable}
        >
            <Box
                display="flex"
                justifyContent="space-between"
                paddingTop={2}
                paddingLeft={3}
                paddingRight={3}
            >
                <Box>
                    <Typography variant="h6">
                        {editMode ? 'Edit' : 'New'} {editedVariable.category === 'terraform' ? 'Terraform' : 'Environment'} Variable
                    </Typography>
                    {editedVariable.metadata && <Typography variant="body2" color="textSecondary">
                        last updated {moment(editedVariable.metadata.updatedAt as moment.MomentInput).fromNow()}
                    </Typography>}
                    {editedVariable.sensitive && editMode && <Chip color="warning" sx={{ mt: 0.5 }} size="xs" label="Sensitive" />}
                </Box>
            </Box>
            <DialogContent>
                {error && <Alert sx={{ marginBottom: 1 }} severity={error.severity}>
                    {error.message}
                </Alert>}
                {!editMode && <FormControlLabel
                    sx={{ mb: 2 }}
                    control={<Checkbox
                        color="warning"
                        checked={editedVariable.sensitive}
                        onChange={onSensitiveChange}
                    />}
                    label={<Box>
                        <Typography variant="body1">Sensitive</Typography>
                        <Typography variant="body2" color="textSecondary">
                            Any variable that contains sensitive information such as passwords or API keys should be marked as sensitive.
                        </Typography>
                    </Box>}
                />}
                <TextField
                    label="Key"
                    name="key"
                    size="small"
                    margin="normal"
                    fullWidth
                    defaultValue={editedVariable.key}
                    onChange={onFieldChange}
                    autoComplete="off"
                    autoFocus={!editedVariable.id}
                />
                <TextField
                    label="Value"
                    name="value"
                    size="small"
                    margin="normal"
                    rows={6}
                    multiline
                    fullWidth
                    defaultValue={editedVariable.value}
                    onChange={onFieldChange}
                    autoComplete="off"
                />
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose} color="inherit">
                    Cancel
                </Button>
                <LoadingButton loading={createInFlight || updateInFlight} onClick={saveVariable} variant="contained">
                    Save
                </LoadingButton>
            </DialogActions>
        </Dialog>
    ) : null;
}

export default EditVariableDialog;
