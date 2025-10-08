import { useState } from 'react';
import { Box, Button, Divider, Typography } from '@mui/material';
import LoadingButton from '@mui/lab/LoadingButton';
import graphql from 'babel-plugin-relay/macro';
import { useFragment, useLazyLoadQuery, useMutation } from "react-relay/hooks";
import { Link as RouterLink, useNavigate, useParams } from 'react-router-dom';
import { MutationError } from '../../common/error';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import FederatedRegistryForm, { FormData } from './FederatedRegistryForm';
import { EditFederatedRegistryFragment_group$key } from './__generated__/EditFederatedRegistryFragment_group.graphql';
import { EditFederatedRegistryMutation } from './__generated__/EditFederatedRegistryMutation.graphql';
import { EditFederatedRegistryQuery } from './__generated__/EditFederatedRegistryQuery.graphql';

interface Props {
    fragmentRef: EditFederatedRegistryFragment_group$key;
}

function EditFederatedRegistry({ fragmentRef }: Props) {
    const federatedRegistryId = useParams<{ id: string }>().id as string;
    const navigate = useNavigate();

    const group = useFragment<EditFederatedRegistryFragment_group$key>(
        graphql`
        fragment EditFederatedRegistryFragment_group on Group
            {
                fullPath
            }
      `,
        fragmentRef
    );

    const queryData = useLazyLoadQuery<EditFederatedRegistryQuery>(
        graphql`
        query EditFederatedRegistryQuery($id: String!) {
            node(id: $id) {
                ... on FederatedRegistry {
                    hostname
                    audience
                }
            }
        }
        `,
        { id: federatedRegistryId }
    );

    if (!queryData.node) {
        return (
            <Box display="flex" justifyContent="center" marginTop={4}>
                <Typography color="textSecondary">Federated registry not found</Typography>
            </Box>
        );
    }

    const [commit, isInFlight] = useMutation<EditFederatedRegistryMutation>(graphql`
        mutation EditFederatedRegistryMutation($input: UpdateFederatedRegistryInput!) {
            updateFederatedRegistry(input: $input) {
                federatedRegistry {
                    id
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const [error, setError] = useState<MutationError>();
    const [formData, setFormData] = useState<FormData>({
        hostname: queryData.node.hostname || '',
        audience: queryData.node.audience || ''
    });

    const onUpdate = () => {
        commit({
            variables: {
                input: {
                    id: federatedRegistryId,
                    hostname: formData.hostname.trim(),
                    audience: formData.audience.trim()
                }
            },
            onCompleted: data => {
                if (data.updateFederatedRegistry.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.updateFederatedRegistry.problems.map((problem: any) => problem.message).join('; ')
                    });
                } else if (!data.updateFederatedRegistry.federatedRegistry) {
                    setError({
                        severity: 'error',
                        message: "Unexpected error occurred"
                    });
                } else {
                    navigate(`../${data.updateFederatedRegistry.federatedRegistry.id}`);
                }
            },
            onError: error => {
                setError({
                    severity: 'error',
                    message: `Unexpected error occurred: ${error.message}`
                });
            }
        });
    };

    const isDisabled = !formData.hostname || !formData.audience;

    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={group.fullPath}
                childRoutes={[
                    { title: "federated registries", path: 'federated_registries' },
                    { title: queryData.node?.hostname || '', path: federatedRegistryId },
                    { title: "edit", path: 'edit' },
                ]}
            />
            <Typography variant="h5" gutterBottom>Edit Federated Registry</Typography>
            <FederatedRegistryForm
                data={formData}
                onChange={(data: FormData) => setFormData(data)}
                error={error}
                sx={{ mt: 3 }}
            />
            <Divider sx={{ my: 3, opacity: 0.6 }} />
            <Box>
                <LoadingButton
                    disabled={isDisabled}
                    loading={isInFlight}
                    variant="outlined"
                    color="primary"
                    sx={{ mr: 2 }}
                    onClick={onUpdate}>
                    Update Federated Registry
                </LoadingButton>
                <Button component={RouterLink} color="inherit" to={`../${federatedRegistryId}`}>Cancel</Button>
            </Box>
        </Box>
    );
}

export default EditFederatedRegistry;
