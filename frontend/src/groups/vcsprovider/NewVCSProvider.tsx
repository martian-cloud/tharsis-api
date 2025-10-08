import { useState } from 'react'
import { Box, Button, Typography } from '@mui/material'
import LoadingButton from '@mui/lab/LoadingButton';
import { Link as RouterLink, useNavigate } from 'react-router-dom';
import { useFragment, useMutation } from 'react-relay';
import { MutationError } from '../../common/error';
import graphql from 'babel-plugin-relay/macro';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import VCSProviderForm, { FormData } from './VCSProviderForm';
import { GetConnections } from './VCSProviderList';
import { NewVCSProviderFragment_group$key } from './__generated__/NewVCSProviderFragment_group.graphql'
import { NewVCSProviderMutation, CreateVCSProviderInput } from './__generated__/NewVCSProviderMutation.graphql'

interface Props {
    fragmentRef: NewVCSProviderFragment_group$key
}

function NewVCSProvider(props: Props) {
    const navigate = useNavigate();

    const group = useFragment<NewVCSProviderFragment_group$key>(
        graphql`
        fragment NewVCSProviderFragment_group on Group
        {
            id
            fullPath
        }
      `,
        props.fragmentRef
    )

    const [commit, isInFlight] = useMutation<NewVCSProviderMutation>(graphql`
        mutation NewVCSProviderMutation($input: CreateVCSProviderInput!, $connections: [ID!]!) {
            createVCSProvider(input: $input) {
                # Use @prependNode to add the node to the connection
                vcsProvider  @prependNode(connections: $connections, edgeTypeName: "VCSProviderEdge")  {
                    id
                    name
                    description
                }
                oAuthAuthorizationUrl
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
        type: undefined,
        name: '',
        url: '',
        description: '',
        oAuthClientId: '',
        oAuthClientSecret: '',
        autoCreateWebhooks: true
    });

    const onSave = () => {
        if (formData.type) {
            const input: CreateVCSProviderInput = {
                name: formData.name,
                description: formData.description,
                groupPath: group.fullPath,
                type: formData.type,
                oAuthClientId: formData.oAuthClientId,
                oAuthClientSecret: formData.oAuthClientSecret,
                autoCreateWebhooks: formData.autoCreateWebhooks
            }
            if (formData.url !== '') {
                input.url = formData.url
            }
            commit({
                variables: {
                    input,
                    connections: GetConnections(group.id)
                },
                onCompleted: data => {
                    if (data.createVCSProvider.problems.length) {
                        setError({
                            severity: 'warning',
                            message: data.createVCSProvider.problems.map((problem: any) => problem.message).join('; ')
                        });
                    } else if (!data.createVCSProvider.vcsProvider) {
                        setError({
                            severity: 'error',
                            message: "Unexpected error occurred"
                        });
                    } else {
                        window.open(data.createVCSProvider.oAuthAuthorizationUrl, '_blank')
                        navigate(`../${data.createVCSProvider.vcsProvider.id}`);
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

    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={group.fullPath}
                childRoutes={[
                    { title: "vcs providers", path: 'vcs_providers' },
                    { title: "new", path: 'new' },
                ]}
            />
            <Typography variant="h5">New VCS Provider</Typography>
            <VCSProviderForm
                data={formData}
                onChange={(data: FormData) => setFormData(data)}
                error={error}
            />
            <Box>
                <LoadingButton
                    loading={isInFlight}
                    disabled={formData.oAuthClientId === '' || formData.oAuthClientSecret === ''}
                    variant="outlined"
                    color="primary"
                    sx={{ marginRight: 2 }}
                    onClick={onSave}>
                    Create VCS Provider</LoadingButton>
                <Button component={RouterLink} color="inherit" to={-1 as any}>Cancel</Button>
            </Box>
        </Box>
    );
}

export default NewVCSProvider
