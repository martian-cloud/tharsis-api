import { useState } from 'react';
import { Button, Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import LoadingButton from '@mui/lab/LoadingButton';
import graphql from 'babel-plugin-relay/macro';
import { MutationError } from '../../../common/error';
import { useFragment, useMutation } from 'react-relay/hooks';
import { GetConnections } from './ManagedIdentityAliasesList';
import ManagedIdentityAliasForm from './ManagedIdentityAliasForm';
import { NewManagedIdentityAliasDialogFragment_managedIdentity$key } from './__generated__/NewManagedIdentityAliasDialogFragment_managedIdentity.graphql';
import { NewManagedIdentityAliasDialogMutation } from './__generated__/NewManagedIdentityAliasDialogMutation.graphql';

export interface FormData {
    name: string
    groupPath: string
}

interface Props {
    onClose: () => void
    fragmentRef: NewManagedIdentityAliasDialogFragment_managedIdentity$key
}

function NewManagedIdentityAliasDialog({ onClose, fragmentRef }: Props) {
    const [error, setError] = useState<MutationError>()
    const [formData, setFormData] = useState<FormData>({
        name: '',
        groupPath: ''
    });

    const managedIdentity = useFragment<NewManagedIdentityAliasDialogFragment_managedIdentity$key>(
        graphql`
        fragment NewManagedIdentityAliasDialogFragment_managedIdentity on ManagedIdentity
        {
            id
            groupPath
        }
    `, fragmentRef);

    const [commit, isInFlight] = useMutation<NewManagedIdentityAliasDialogMutation>(graphql`
        mutation NewManagedIdentityAliasDialogMutation($input: CreateManagedIdentityAliasInput!, $connections: [ID!]!) {
            createManagedIdentityAlias(input: $input) {
                managedIdentity @prependNode(connections: $connections, edgeTypeName: "ManagedIdentityEdge") {
                    id
                    ...ManagedIdentityAliasesListItemFragment_managedIdentity
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const onCreate = () => {
        commit({
            variables: {
                input: {
                    name: formData.name,
                    aliasSourceId: managedIdentity.id,
                    groupPath: formData.groupPath
                },
                connections: GetConnections(managedIdentity.id),
            },
            onCompleted: data => {
                if (data.createManagedIdentityAlias.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.createManagedIdentityAlias.problems.map(problem => problem.message).join('; ')
                    });
                } else if (!data.createManagedIdentityAlias.managedIdentity) {
                    setError({
                        severity: 'error',
                        message: "Unexpected error occurred"
                    });
                } else {
                    onClose()
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

    return (
        <Dialog
            fullWidth
            maxWidth="sm"
            open>
            <DialogTitle>
                New Alias
            </DialogTitle>
            <DialogContent dividers>
                <ManagedIdentityAliasForm
                    data={formData}
                    error={error}
                    onChange={setFormData}
                    groupPath={managedIdentity.groupPath}
                />
            </DialogContent>
            <DialogActions>
                <Button
                    size="small"
                    variant="outlined"
                    color="inherit"
                    onClick={onClose}>Cancel
                </Button>
                <LoadingButton
                    sx={{ marginLeft: 2 }}
                    disabled={formData.name === '' || formData.groupPath === ''}
                    loading={isInFlight}
                    size="small"
                    variant="contained"
                    color="primary"
                    onClick={() => onCreate()}>Create Alias
                </LoadingButton>
            </DialogActions>
        </Dialog>
    );
}

export default NewManagedIdentityAliasDialog
