import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Divider from '@mui/material/Divider';
import Typography from '@mui/material/Typography';
import graphql from 'babel-plugin-relay/macro';
import moment from 'moment';
import { nanoid } from 'nanoid';
import { useState } from 'react';
import { useFragment, useLazyLoadQuery, useMutation } from "react-relay/hooks";
import { Link as RouterLink, useNavigate, useParams } from 'react-router-dom';
import { MutationError } from '../../common/error';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import ClientCredentialsDialog from './ClientCredentialsDialog';
import ServiceAccountForm, { FormData } from './ServiceAccountForm';
import { EditServiceAccountFragment_group$key } from './__generated__/EditServiceAccountFragment_group.graphql';
import { EditServiceAccountMutation } from './__generated__/EditServiceAccountMutation.graphql';
import { EditServiceAccountQuery } from './__generated__/EditServiceAccountQuery.graphql';

interface Props {
    fragmentRef: EditServiceAccountFragment_group$key
}

function EditServiceAccount(props: Props) {
    const { id } = useParams();
    const navigate = useNavigate();

    const serviceAccountId = id as string;

    const group = useFragment<EditServiceAccountFragment_group$key>(
        graphql`
        fragment EditServiceAccountFragment_group on Group
        {
          id
          fullPath
        }
      `, props.fragmentRef
    );

    const queryData = useLazyLoadQuery<EditServiceAccountQuery>(graphql`
        query EditServiceAccountQuery($id: String!) {
            serviceAccount(id: $id) {
                id
                name
                description
                resourcePath
                createdBy
                clientCredentialsEnabled
                clientSecretExpiresAt
                oidcTrustPolicies {
                    issuer
                    boundClaimsType
                    boundClaims {
                        name
                        value
                    }
                }
            }
        }
    `, { id: serviceAccountId })

    const [commit, isInFlight] = useMutation<EditServiceAccountMutation>(graphql`
        mutation EditServiceAccountMutation($input: UpdateServiceAccountInput!) {
            updateServiceAccount(input: $input) {
                serviceAccount {
                    id
                    name
                    description
                    resourcePath
                    createdBy
                    clientCredentialsEnabled
                    clientSecretExpiresAt
                    oidcTrustPolicies {
                        issuer
                        boundClaimsType
                        boundClaims {
                            name
                            value
                        }
                    }
                }
                clientSecret
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const [error, setError] = useState<MutationError>()
    const [clientCredentials, setClientCredentials] = useState<{ clientId: string; clientSecret: string; expiresAt: string } | null>(null);
    const [formData, setFormData] = useState<FormData | null>(() => queryData.serviceAccount ? {
        name: queryData.serviceAccount.name,
        description: queryData.serviceAccount.description,
        oidcTrustPolicies: queryData.serviceAccount.oidcTrustPolicies.map(trustPolicy => ({ ...trustPolicy, _id: nanoid() })),
        enableClientCredentials: queryData.serviceAccount.clientCredentialsEnabled,
        clientSecretExpiresAt: queryData.serviceAccount.clientSecretExpiresAt ? moment(queryData.serviceAccount.clientSecretExpiresAt) : null
    } : null);

    const onUpdate = () => {
        if (formData) {
            commit({
                variables: {
                    input: {
                        id: serviceAccountId,
                        description: formData.description,
                        oidcTrustPolicies: formData.oidcTrustPolicies,
                        enableClientCredentials: formData.enableClientCredentials,
                        clientSecretExpiresAt: formData.enableClientCredentials ? formData.clientSecretExpiresAt?.toISOString() : undefined
                    }
                },
                onCompleted: data => {
                    if (data.updateServiceAccount.problems.length) {
                        setError({
                            severity: 'warning',
                            message: data.updateServiceAccount.problems.map((problem: any) => problem.message).join('; ')
                        });
                    } else if (!data.updateServiceAccount.serviceAccount) {
                        setError({
                            severity: 'error',
                            message: "Unexpected error occurred"
                        });
                    } else {
                        if (data.updateServiceAccount.clientSecret) {
                            setClientCredentials({
                                clientId: data.updateServiceAccount.serviceAccount.id,
                                clientSecret: data.updateServiceAccount.clientSecret,
                                expiresAt: data.updateServiceAccount.serviceAccount.clientSecretExpiresAt
                            });
                        } else {
                            navigate(`../${data.updateServiceAccount.serviceAccount.id}`);
                        }
                    }
                },
                onError: error => {
                    setError({
                        severity: 'error',
                        message: `Unexpected error occurred: ${error.message}`
                    });
                }
            });
        }
    };

    const onCredentialsDialogClose = () => {
        setClientCredentials(null);
        navigate(`../${serviceAccountId}`);
    };

    return formData ? (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={group.fullPath}
                childRoutes={[
                    { title: "service accounts", path: 'service_accounts' },
                    { title: formData.name, path: serviceAccountId },
                    { title: "edit", path: 'edit' },
                ]}
            />
            <Typography variant="h5">Edit Service Account</Typography>
            <ServiceAccountForm
                editMode
                data={formData}
                onChange={setFormData}
                error={error}
            />
            <Divider sx={{ marginTop: 4 }} />
            <Box marginTop={2}>
                <Button
                    loading={isInFlight}
                    variant="outlined"
                    color="primary"
                    sx={{ marginRight: 2 }}
                    onClick={onUpdate}
                >
                    Update Service Account
                </Button>
                <Button component={RouterLink} color="inherit" to={-1 as any}>Cancel</Button>
            </Box>
            {clientCredentials && (
                <ClientCredentialsDialog
                    clientId={clientCredentials.clientId}
                    clientSecret={clientCredentials.clientSecret}
                    onClose={onCredentialsDialogClose}
                />
            )}
        </Box>
    ) : <Box>Service Account Not found</Box>;
}

export default EditServiceAccount;
