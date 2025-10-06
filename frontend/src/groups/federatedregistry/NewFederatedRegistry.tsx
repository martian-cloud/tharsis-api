import { useState } from 'react';
import { Box, Button, Divider, Typography } from '@mui/material';
import LoadingButton from '@mui/lab/LoadingButton';
import graphql from 'babel-plugin-relay/macro';
import { useFragment, useMutation } from "react-relay/hooks";
import { Link as RouterLink, useNavigate } from 'react-router-dom';
import { MutationError } from '../../common/error';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import FederatedRegistryForm, { FormData } from './FederatedRegistryForm';
import { GetConnections } from './FederatedRegistryList';
import { NewFederatedRegistryFragment_group$key } from './__generated__/NewFederatedRegistryFragment_group.graphql';
import { NewFederatedRegistryMutation } from './__generated__/NewFederatedRegistryMutation.graphql';

interface Props {
    fragmentRef: NewFederatedRegistryFragment_group$key;
}

function NewFederatedRegistry({ fragmentRef }: Props) {
    const navigate = useNavigate();

    const group = useFragment<NewFederatedRegistryFragment_group$key>(
        graphql`
        fragment NewFederatedRegistryFragment_group on Group
        {
          id
          fullPath
        }
      `,
        fragmentRef
    );

    const [commit, isInFlight] = useMutation<NewFederatedRegistryMutation>(graphql`
        mutation NewFederatedRegistryMutation($input: CreateFederatedRegistryInput!, $connections: [ID!]!) {
            createFederatedRegistry(input: $input) {
                federatedRegistry @prependNode(connections: $connections, edgeTypeName: "FederatedRegistryEdge") {
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
        hostname: '',
        audience: ''
    });

    const onSave = () => {
        commit({
            variables: {
                input: {
                    groupPath: group.fullPath,
                    hostname: formData.hostname.trim(),
                    audience: formData.audience.trim()
                },
                connections: GetConnections(group.id)
            },
            onCompleted: data => {
                if (data.createFederatedRegistry.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.createFederatedRegistry.problems.map((problem: any) => problem.message).join('; ')
                    });
                } else if (!data.createFederatedRegistry.federatedRegistry) {
                    setError({
                        severity: 'error',
                        message: "Unexpected error occurred"
                    });
                } else {
                    navigate(`../${data.createFederatedRegistry.federatedRegistry.id}`);
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
                    { title: "new", path: 'new' },
                ]}
            />
            <Typography variant="h5" gutterBottom>New Federated Registry</Typography>
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
                    onClick={onSave}>
                    Create Federated Registry
                </LoadingButton>
                <Button component={RouterLink} color="inherit" to={-1 as any}>Cancel</Button>
            </Box>
        </Box>
    );
}

export default NewFederatedRegistry;
