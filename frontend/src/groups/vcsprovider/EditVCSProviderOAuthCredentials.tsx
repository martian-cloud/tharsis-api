import { useState } from 'react'
import { Box, Button, Typography } from "@mui/material";
import LoadingButton from '@mui/lab/LoadingButton';
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import { Link as RouterLink, useParams } from 'react-router-dom';
import { useFragment, useLazyLoadQuery, useMutation } from "react-relay";
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import { MutationError } from '../../common/error';
import VCSProviderOAuthCredForm, { OAuthFormData } from './VCSProviderOAuthCredForm';
import { EditVCSProviderOAuthCredentialsFragment_group$key } from './__generated__/EditVCSProviderOAuthCredentialsFragment_group.graphql';
import { EditVCSProviderOAuthCredentialsQuery } from './__generated__/EditVCSProviderOAuthCredentialsQuery.graphql';
import { EditVCSProviderOAuthCredentialsMutation } from './__generated__/EditVCSProviderOAuthCredentialsMutation.graphql';

interface Props {
    fragmentRef: EditVCSProviderOAuthCredentialsFragment_group$key
}

function EditVCSProviderOAuth(props: Props) {
    const { id } = useParams();
    const { enqueueSnackbar } = useSnackbar();

    const vcsProviderId = id as string;

    const group = useFragment<EditVCSProviderOAuthCredentialsFragment_group$key>(
        graphql`
        fragment EditVCSProviderOAuthCredentialsFragment_group on Group
        {
            id
            fullPath
        }
        `, props.fragmentRef
    );

    const queryData = useLazyLoadQuery<EditVCSProviderOAuthCredentialsQuery>(graphql`
        query EditVCSProviderOAuthCredentialsQuery($id: String!) {
            node(id: $id) {
                ... on VCSProvider {
                    name
                    type
                }
            }
        }
    `, { id: vcsProviderId });

    const [commit, isInFlight] = useMutation<EditVCSProviderOAuthCredentialsMutation>(graphql`
        mutation EditVCSProviderOAuthCredentialsMutation($input: UpdateVCSProviderInput!) {
            updateVCSProvider(input: $input) {
                vcsProvider{
                    id
                    name
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
    const [formData, setFormData] = useState<OAuthFormData>({
        oAuthClientId: '',
        oAuthClientSecret: '',
    });

    const onUpdate = () => {
        commit({
            variables: {
                input: {
                    id: vcsProviderId,
                    oAuthClientId: formData.oAuthClientId,
                    oAuthClientSecret: formData.oAuthClientSecret
                }
            },
            onCompleted: data => {
                if (data.updateVCSProvider.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.updateVCSProvider.problems.map((problem: any) => problem.message).join('; ')
                    });
                } else if (!data.updateVCSProvider.vcsProvider) {
                    setError({
                        severity: 'error',
                        message: "Unexpected error occurred"
                    });
                }
                else {
                    enqueueSnackbar('OAuth credentials updated', { variant: 'success' })
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

    const vcsProvider = queryData.node as any

    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={group.fullPath}
                childRoutes={[
                    { title: "vcs providers", path: 'vcs_providers' },
                    { title: vcsProvider.name, path: vcsProviderId },
                    { title: "edit oauth credentials", path: 'edit_oauth_credentials' },
                ]}
            />
            <Typography variant="h5">Edit OAuth Credentials</Typography>
            <VCSProviderOAuthCredForm
                data={formData}
                onChange={(data: OAuthFormData) => setFormData(data)}
                error={error}
            />
            <Box marginTop={2}>
                <LoadingButton
                    loading={isInFlight}
                    disabled={formData.oAuthClientId === '' || formData.oAuthClientSecret === ''}
                    variant="outlined"
                    color="primary"
                    sx={{ marginRight: 2 }}
                    onClick={onUpdate}>
                    Update OAuth Credentials
                </LoadingButton>
                <Button
                    component={RouterLink}
                    color="inherit"
                    to={-1 as any}>
                    Cancel
                </Button>
            </Box>
        </Box>
    );
}

export default EditVCSProviderOAuth
