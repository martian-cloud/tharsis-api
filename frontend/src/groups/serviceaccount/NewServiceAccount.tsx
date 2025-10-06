import LoadingButton from '@mui/lab/LoadingButton';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Divider from '@mui/material/Divider';
import Typography from '@mui/material/Typography';
import graphql from 'babel-plugin-relay/macro';
import { nanoid } from 'nanoid';
import { useState } from 'react';
import { useFragment, useMutation } from "react-relay/hooks";
import { Link as RouterLink, useNavigate } from 'react-router-dom';
import { MutationError } from '../../common/error';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import ServiceAccountForm, { FormData } from './ServiceAccountForm';
import { GetConnections } from './ServiceAccountList';
import { NewServiceAccountFragment_group$key } from './__generated__/NewServiceAccountFragment_group.graphql';
import { NewServiceAccountMutation } from './__generated__/NewServiceAccountMutation.graphql';

interface Props {
    fragmentRef: NewServiceAccountFragment_group$key
}

function NewServiceAccount(props: Props) {
    const navigate = useNavigate();

    const group = useFragment<NewServiceAccountFragment_group$key>(
        graphql`
        fragment NewServiceAccountFragment_group on Group
        {
          id
          fullPath
        }
      `,
        props.fragmentRef
    );

    const [commit, isInFlight] = useMutation<NewServiceAccountMutation>(graphql`
        mutation NewServiceAccountMutation($input: CreateServiceAccountInput!, $connections: [ID!]!) {
            createServiceAccount(input: $input) {
                # Use @prependNode to add the node to the connection
                serviceAccount  @prependNode(connections: $connections, edgeTypeName: "ServiceAccountEdge")  {
                    id
                    name
                    description
                    resourcePath
                    createdBy
                    oidcTrustPolicies {
                        issuer
                        boundClaimsType
                        boundClaims {
                            name
                            value
                        }
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

    const [error, setError] = useState<MutationError>()
    const [formData, setFormData] = useState<FormData>({
        name: '',
        description: '',
        oidcTrustPolicies: [{ _id: nanoid(), issuer: '', boundClaimsType: 'STRING', boundClaims: [] }]
    });

    const onSave = () => {
        commit({
            variables: {
                input: {
                    groupPath: group.fullPath,
                    name: formData.name,
                    description: formData.description,
                    oidcTrustPolicies: formData.oidcTrustPolicies
                },
                connections: GetConnections(group.id)
            },
            onCompleted: data => {
                if (data.createServiceAccount.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.createServiceAccount.problems.map((problem: any) => problem.message).join('; ')
                    });
                } else if (!data.createServiceAccount.serviceAccount) {
                    setError({
                        severity: 'error',
                        message: "Unexpected error occurred"
                    });
                } else {
                    navigate(`../${data.createServiceAccount.serviceAccount.id}`);
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

    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={group.fullPath}
                childRoutes={[
                    { title: "service accounts", path: 'service_accounts' },
                    { title: "new", path: 'new' },
                ]}
            />
            <Typography variant="h5">New Service Account</Typography>
            <ServiceAccountForm
                data={formData}
                onChange={(data: FormData) => setFormData(data)}
                error={error}
            />
            <Divider sx={{ opacity: 0.6 }} />
            <Box marginTop={2}>
                <LoadingButton
                    loading={isInFlight}
                    variant="outlined"
                    color="primary"
                    sx={{ marginRight: 2 }}
                    onClick={onSave}>
                    Create Service Account
                </LoadingButton>
                <Button component={RouterLink} color="inherit" to={-1 as any}>Cancel</Button>
            </Box>
        </Box>
    );
}

export default NewServiceAccount;
