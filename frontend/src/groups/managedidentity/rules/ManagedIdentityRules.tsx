import { LoadingButton } from '@mui/lab';
import { Box, Button, Dialog, DialogActions, DialogContent, DialogTitle, Paper, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import { useState } from 'react';
import { useFragment, useMutation } from 'react-relay';
import { RecordSourceProxy } from 'relay-runtime';
import { MutationError } from '../../../common/error';
import EditManagedIdentityRuleDialog from './EditManagedIdentityRuleDialog';
import ManagedIdentityRulesList from './ManagedIdentityRulesList';
import NewManagedIdentityRuleDialog from './NewManagedIdentityRuleDialog';
import { ManagedIdentityRulesCreateRuleMutation, ManagedIdentityRulesCreateRuleMutation$data } from './__generated__/ManagedIdentityRulesCreateRuleMutation.graphql';
import { ManagedIdentityRulesDeleteMutation, ManagedIdentityRulesDeleteMutation$data } from './__generated__/ManagedIdentityRulesDeleteMutation.graphql';
import { ManagedIdentityRulesFragment_managedIdentity$key } from './__generated__/ManagedIdentityRulesFragment_managedIdentity.graphql';
import { ManagedIdentityRulesUpdateRuleMutation } from './__generated__/ManagedIdentityRulesUpdateRuleMutation.graphql';

interface ConfirmationDialogProps {
    deleteInProgress: boolean;
    onClose: (confirm?: boolean) => void
}

function DeleteConfirmationDialog(props: ConfirmationDialogProps) {
    const { deleteInProgress, onClose, ...other } = props;
    return (
        <Dialog
            maxWidth="xs"
            open
            {...other}
        >
            <DialogTitle>Delete Rule</DialogTitle>
            <DialogContent dividers>
                Are you sure you want to delete this access rule?
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
    fragmentRef: ManagedIdentityRulesFragment_managedIdentity$key;
    groupPath: string
}

function ManagedIdentityRules(props: Props) {
    const { groupPath } = props;

    const data = useFragment<ManagedIdentityRulesFragment_managedIdentity$key>(
        graphql`
        fragment ManagedIdentityRulesFragment_managedIdentity on ManagedIdentity
        {
            id
            isAlias
            accessRules {
                id
                type
                runStage
                moduleAttestationPolicies {
                    publicKey
                    predicateType
                }
                allowedUsers {
                    id
                    username
                    email
                }
                allowedTeams {
                    id
                    name
                }
                allowedServiceAccounts {
                    id
                    name
                    resourcePath
                }
            }
        }
    `, props.fragmentRef);

    const [commitDeleteRule, commitDeleteRuleInFlight] = useMutation<ManagedIdentityRulesDeleteMutation>(graphql`
        mutation ManagedIdentityRulesDeleteMutation($input: DeleteManagedIdentityAccessRuleInput! ) {
            deleteManagedIdentityAccessRule(input: $input) {
                accessRule {
                    id
                }
                problems {
                    message
                    field
                    type
                }
            }
    }`);

    const [commitCreateRule, commitCreateRuleInFlight] = useMutation<ManagedIdentityRulesCreateRuleMutation>(graphql`
        mutation ManagedIdentityRulesCreateRuleMutation($input: CreateManagedIdentityAccessRuleInput!) {
            createManagedIdentityAccessRule(input: $input) {
                accessRule {
                    id
                    type
                    runStage
                    allowedUsers {
                        id
                        username
                        email
                    }
                    allowedTeams {
                        id
                        name
                    }
                    allowedServiceAccounts {
                        id
                        resourcePath
                    }
                    moduleAttestationPolicies {
                        publicKey
                        predicateType
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

    const [commitUpdateRule, commitUpdateRuleInFlight] = useMutation<ManagedIdentityRulesUpdateRuleMutation>(graphql`
    mutation ManagedIdentityRulesUpdateRuleMutation($input: UpdateManagedIdentityAccessRuleInput!) {
        updateManagedIdentityAccessRule(input: $input) {
                accessRule {
                    id
                    type
                    runStage
                    allowedUsers {
                        id
                        username
                        email
                    }
                    allowedTeams {
                        id
                        name
                    }
                    allowedServiceAccounts {
                        id
                        name
                        resourcePath
                    }
                    moduleAttestationPolicies {
                        publicKey
                        predicateType
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

    const { enqueueSnackbar } = useSnackbar();
    const [ruleToDelete, setRuleToDelete] = useState<any>(null);
    const [ruleToEdit, setRuleToEdit] = useState<any>(null);
    const [showCreateNewRuleDialog, setShowCreateNewRuleDialog] = useState(false);
    const [error, setError] = useState<MutationError>();

    const onDeleteConfirmationDialogClosed = (confirm?: boolean) => {
        if (confirm) {
            commitDeleteRule({
                variables: {
                    input: {
                        id: ruleToDelete.id
                    },
                },
                onCompleted: data => {
                    setRuleToDelete(null);

                    if (data.deleteManagedIdentityAccessRule.problems.length) {
                        enqueueSnackbar(data.deleteManagedIdentityAccessRule.problems.map(problem => problem.message).join('; '), { variant: 'warning' });
                    }
                },
                onError: error => {
                    setRuleToDelete(null);
                    enqueueSnackbar(`Unexpected error occurred: ${error.message}`, { variant: 'error' });
                },
                updater: (store: RecordSourceProxy, payload: ManagedIdentityRulesDeleteMutation$data | null | undefined) => {
                    if (!payload || !payload.deleteManagedIdentityAccessRule.accessRule) {
                        return;
                    }

                    const managedIdentityRecord = store.get(data.id);
                    if (managedIdentityRecord == null) {
                        return;
                    }

                    const ruleRecord = store.get(payload.deleteManagedIdentityAccessRule.accessRule.id);
                    if (ruleRecord == null) {
                        return;
                    }

                    const rules = managedIdentityRecord.getLinkedRecords('accessRules') || [];
                    const index = rules.findIndex(rule => rule.getValue('id') === ruleToDelete.id)
                    if (index !== -1) {
                        const rulesCopy = [...rules];
                        rulesCopy.splice(index, 1)
                        managedIdentityRecord.setLinkedRecords(rulesCopy, 'accessRules')
                    }
                }
            });
        } else {
            setRuleToDelete(null);
        }
    };

    const onCreateRule = (rule: any) => {
        commitCreateRule({
            variables: {
                input: {
                    managedIdentityId: data.id,
                    type: rule.type,
                    runStage: rule.runStage,
                    allowedServiceAccounts: rule.allowedServiceAccounts.map((sa: any) => (sa.resourcePath)) || [],
                    allowedUsers: rule.allowedUsers.map((user: any) => (user.username)) || [],
                    allowedTeams: rule.allowedTeams.map((team: any) => (team.name)) || [],
                    moduleAttestationPolicies: rule.moduleAttestationPolicies.map((att: any) => ({...att, predicateType: att.predicateType === '' ? undefined : att.predicateType}))
                }
            },
            onCompleted: data => {
                if (data.createManagedIdentityAccessRule.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.createManagedIdentityAccessRule.problems.map(problem => problem.message).join('; ')
                    });
                } else if (!data.createManagedIdentityAccessRule) {
                    setError({
                        severity: 'error',
                        message: "Unexpected error occurred"
                    });
                } else {
                    setShowCreateNewRuleDialog(false);
                }
            },
            onError: error => {
                setError({
                    severity: 'error',
                    message: `Unexpected error occurred: ${error.message}`
                });
            },
            updater: (store: RecordSourceProxy, payload: ManagedIdentityRulesCreateRuleMutation$data | null | undefined) => {
                if (!payload || !payload.createManagedIdentityAccessRule.accessRule) {
                    return;
                }

                const managedIdentityRecord = store.get(data.id);
                if (managedIdentityRecord == null) {
                    return;
                }

                const ruleRecord = store.get(payload.createManagedIdentityAccessRule.accessRule.id);
                if (ruleRecord == null) {
                    return;
                }

                const rules = managedIdentityRecord.getLinkedRecords('accessRules') || [];
                managedIdentityRecord.setLinkedRecords([...rules, ruleRecord], 'accessRules')
            }
        })
    };

    const onUpdateRule = (rule: any) => {
        setError(undefined);
        commitUpdateRule({
            variables: {
                input: {
                    id: rule.id,
                    runStage: rule.runStage,
                    allowedServiceAccounts: rule.allowedServiceAccounts.map((sa: any) => (sa.resourcePath)) || [],
                    allowedUsers: rule.allowedUsers.map((user: any) => (user.username)) || [],
                    allowedTeams: rule.allowedTeams.map((team: any) => (team.name)) || [],
                    moduleAttestationPolicies: rule.moduleAttestationPolicies.map((att: any) => ({...att, predicateType: att.predicateType === '' ? undefined : att.predicateType}))
                },
            },
            onCompleted: data => {
                if (data.updateManagedIdentityAccessRule.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.updateManagedIdentityAccessRule.problems.map(problem => problem.message).join('; ')
                    });
                } else {
                    setRuleToEdit(null);
                }
            },
            onError: error => {
                setError({
                    severity: 'error',
                    message: `Unexpected Error Occurred: ${error.message}`
                })
            }
        })
    };

    return (
        <Box>
            <Typography sx={{ marginBottom: 2 }} color="textSecondary">
                Access rules contain policies that control if this managed identity can be used for a particular run stage
            </Typography>
            {data.accessRules.length > 0 && <ManagedIdentityRulesList
                accessRules={data.accessRules}
                isAlias={data.isAlias}
                onEdit={setRuleToEdit}
                onDelete={setRuleToDelete}
            />}
            {data.accessRules.length === 0 && <Paper sx={{ p: 2 }}>
                <Typography>No rules exist for this managed identity</Typography>
            </Paper>}
            {!data.isAlias && <Button
                sx={{ marginTop: 3 }}
                color="secondary"
                size="small"
                variant="outlined"
                onClick={() => setShowCreateNewRuleDialog(true)}>
                Add Access Rule
            </Button>}
            {ruleToEdit && <EditManagedIdentityRuleDialog
                inputRule={ruleToEdit}
                groupPath={groupPath}
                submitInProgress={commitUpdateRuleInFlight}
                error={error}
                onSubmit={onUpdateRule}
                onClose={() => {
                    setRuleToEdit(null);
                    setError(undefined);
                }} />}
            {showCreateNewRuleDialog && <NewManagedIdentityRuleDialog
                groupPath={groupPath}
                submitInProgress={commitCreateRuleInFlight}
                error={error}
                onSubmit={onCreateRule}
                onClose={() => {
                    setShowCreateNewRuleDialog(false);
                    setError(undefined);
                }}
            />}
            {ruleToDelete && <DeleteConfirmationDialog
                deleteInProgress={commitDeleteRuleInFlight}
                onClose={onDeleteConfirmationDialogClosed}
            />}
        </Box>
    );
}

export default ManagedIdentityRules
