import { LoadingButton } from '@mui/lab';
import { Dialog, DialogActions, DialogContent, DialogTitle, useTheme } from '@mui/material';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import { darken } from '@mui/material/styles';
import TextField from '@mui/material/TextField';
import ToggleButton from '@mui/material/ToggleButton';
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup';
import Typography from '@mui/material/Typography';
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import React, { useState } from 'react';
import { useFragment, useMutation } from 'react-relay/hooks';
import { Route, Routes } from 'react-router-dom';
import NamespaceBreadcrumbs from '../NamespaceBreadcrumbs';
import EditVariableDialog from './EditVariableDialog';
import VariableList from './VariableList';
import { VariablesDeleteVariableMutation } from './__generated__/VariablesDeleteVariableMutation.graphql';
import { VariablesFragment_variables$key } from './__generated__/VariablesFragment_variables.graphql';
import VariableHistoryDialog from './VariableHistoryDialog';

interface ConfirmationDialogProps {
    variable: any
    deleteInProgress: boolean;
    keepMounted: boolean;
    onClose: (confirm?: boolean) => void
}

function DeleteConfirmationDialog(props: ConfirmationDialogProps) {
    const { variable, deleteInProgress, onClose, ...other } = props;
    return (
        <Dialog
            maxWidth="xs"
            open={!!variable}
            {...other}
        >
            <DialogTitle>Delete Variable</DialogTitle>
            <DialogContent dividers>
                Are you sure you want to delete the variable <strong>{variable?.key}</strong>?
            </DialogContent>
            <DialogActions>
                <Button color="inherit" onClick={() => onClose()}>
                    Cancel
                </Button>
                <LoadingButton color="error" loading={deleteInProgress} onClick={() => onClose(true)}>Delete</LoadingButton>
            </DialogActions>
        </Dialog>
    );
}

interface Props {
    fragmentRef: VariablesFragment_variables$key
}

const variableSearchFilter = (search: string) => (variable: any) => {
    return variable.key.toLowerCase().startsWith(search);
}

function Variables(props: Props) {
    const theme = useTheme();
    const { enqueueSnackbar } = useSnackbar();

    const data = useFragment<VariablesFragment_variables$key>(
        graphql`
        fragment VariablesFragment_variables on Namespace
        {
            id
            fullPath
            variables {
                id
                key
                category
                ...VariableListItemFragment_variable
            }
        }
      `, props.fragmentRef);

    const [commitDeleteVariable, commitInFlight] = useMutation<VariablesDeleteVariableMutation>(graphql`
        mutation VariablesDeleteVariableMutation($input: DeleteNamespaceVariableInput!) {
            deleteNamespaceVariable(input: $input) {
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

    const [showValues, setShowValues] = useState(false);
    const [variableCategory, setVariableCategory] = useState('terraform');
    const [search, setSearch] = useState('');
    const [variableToEdit, setVariableToEdit] = useState<any>(null);
    const [variableToDelete, setVariableToDelete] = useState<any>(null);
    const [variableToShowHistory, setVariableToShowHistory] = useState<any>(null);

    const onEditVariable = (variable: any) => {
        setVariableToEdit(variable);
    };

    const onNewVariable = () => {
        setVariableToEdit({
            key: '',
            value: '',
            category: variableCategory
        });
    };

    const onVariableCategoryChange = (event: React.MouseEvent<HTMLElement>, newCategory: string) => {
        if (newCategory) {
            setVariableCategory(newCategory);
        }
    };

    const onSearchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        setSearch(event.target.value.toLowerCase());
    };

    const onCloseDeleteVariableConfirmation = (confirm?: boolean) => {
        if (confirm) {
            commitDeleteVariable({
                variables: {
                    input: {
                        id: variableToDelete.id
                    },
                },
                onCompleted: data => {
                    if (data.deleteNamespaceVariable.problems.length) {
                        enqueueSnackbar(data.deleteNamespaceVariable.problems.map(problem => problem.message).join('; '), { variant: 'warning' });
                    }
                    setVariableToDelete(null);
                },
                onError: error => {
                    setVariableToDelete(null);
                    enqueueSnackbar(`Unexpected error occurred: ${error.message}`, { variant: 'error' });
                }
            });
        } else {
            setVariableToDelete(null);
        }
    };

    const variables = data.variables.filter((v: any) => v.category === variableCategory);
    const filteredVariables = search ? variables.filter(variableSearchFilter(search)) : variables;

    return (
        <Box>
            <Routes>
                <Route index element={<Box>
                    <NamespaceBreadcrumbs
                        namespacePath={data.fullPath}
                        childRoutes={[
                            { title: "variables", path: 'variables' }
                        ]}
                    />
                    <Box sx={{
                        marginBottom: 4,
                        display: 'flex',
                        flexDirection: 'row',
                        justifyContent: 'space-between',
                        [theme.breakpoints.down('lg')]: {
                            flexDirection: 'column',
                            alignItems: 'flex-start',
                            '& > *:not(:last-child)': {
                                marginBottom: 2
                            },
                        }
                    }}>
                        <ToggleButtonGroup
                            size="small"
                            color="primary"
                            value={variableCategory}
                            exclusive
                            onChange={onVariableCategoryChange}
                            sx={{ height: '100%' }}
                        >
                            <ToggleButton value="terraform" size="small">Terraform</ToggleButton>
                            <ToggleButton value="environment" size="small">Environment</ToggleButton>
                        </ToggleButtonGroup>
                        <Stack direction="row" spacing={2}>
                            <TextField
                                size="small"
                                margin='none'
                                placeholder="search for variables"
                                InputProps={{
                                    sx: { background: darken(theme.palette.background.default, 0.5) }
                                }}
                                sx={{ width: 300 }}
                                onChange={onSearchChange}
                                autoComplete="off"
                            />
                            <Button
                                size="small"
                                color="info"
                                sx={{ height: '100%' }}
                                onClick={() => setShowValues(!showValues)}
                            >
                                {showValues ? 'Hide Values' : 'Show Values'}
                            </Button>
                        </Stack>
                    </Box>
                    {(filteredVariables.length === 0 && search !== '') && <Typography sx={{ padding: 2 }} align="center" color="textSecondary">
                        No variables matching search
                    </Typography>}
                    {search === '' && filteredVariables.length === 0 && <Paper variant="outlined" sx={{ marginTop: 4, display: 'flex', justifyContent: 'center', marginBottom: 6 }}>
                        <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center">
                            {variableCategory === 'terraform' && <Typography color="textSecondary" align="center" sx={{ marginBottom: 2 }}>
                                Add Terraform variables which will be provided as inputs to your Terraform modules
                            </Typography>}
                            {variableCategory === 'environment' && <Typography color="textSecondary" align="center" sx={{ marginBottom: 2 }}>
                                Add environment variables which will be automatically set when executing Terraform runs
                            </Typography>}
                            <Button variant="outlined" color="primary" onClick={onNewVariable}>New Variable</Button>
                        </Box>
                    </Paper>}
                    {(filteredVariables.length > 0) && <Box marginBottom={6}>
                        <Paper>
                            <Box padding={2} display="flex" alignItems="center" justifyContent="space-between">
                                <Typography variant="subtitle1">{filteredVariables.length} variable{filteredVariables.length === 1 ? '' : 's'}</Typography>
                                <Button size="small" variant="outlined" color="secondary" onClick={onNewVariable}>New Variable</Button>
                            </Box>
                        </Paper>
                        <VariableList
                            namespacePath={data.fullPath}
                            variables={filteredVariables}
                            showValues={showValues}
                            onEditVariable={onEditVariable}
                            onDeleteVariable={(variable: any) => setVariableToDelete(variable)}
                            onShowHistory={(variable: any) => setVariableToShowHistory(variable)}
                        />
                    </Box>}
                </Box>} />
            </Routes>
            {variableToShowHistory && <VariableHistoryDialog
                variableId={variableToShowHistory.id}
                sensitive={variableToShowHistory.sensitive}
                onClose={() => setVariableToShowHistory(null)}
            />}
            {variableToEdit && <EditVariableDialog variable={variableToEdit} namespacePath={data.fullPath} onClose={() => setVariableToEdit(null)} />}
            <DeleteConfirmationDialog
                keepMounted
                variable={variableToDelete}
                deleteInProgress={commitInFlight}
                onClose={onCloseDeleteVariableConfirmation}
            />
        </Box>
    );
}

export default Variables;
