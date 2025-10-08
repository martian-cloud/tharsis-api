import { useState } from 'react'
import { Box, Button, Typography } from "@mui/material";
import LoadingButton from '@mui/lab/LoadingButton';
import { Link as RouterLink, useNavigate, useParams } from 'react-router-dom';
import { MutationError } from '../../common/error';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import graphql from 'babel-plugin-relay/macro';
import { useFragment, useLazyLoadQuery, useMutation } from "react-relay/hooks";
import VCSProviderForm, { FormData } from "./VCSProviderForm";
import { EditVCSProviderFragment_group$key } from './__generated__/EditVCSProviderFragment_group.graphql'
import { EditVCSProviderQuery } from './__generated__/EditVCSProviderQuery.graphql'
import { EditVCSProviderMutation } from './__generated__/EditVCSProviderMutation.graphql';

interface Props {
    fragmentRef: EditVCSProviderFragment_group$key
}

function EditVCSProvider(props: Props) {
    const { id } = useParams();
    const navigate = useNavigate();

    const vcsProviderId = id as string;

    const group = useFragment<EditVCSProviderFragment_group$key>(
        graphql`
        fragment EditVCSProviderFragment_group on Group
        {
          id
          fullPath
        }
      `, props.fragmentRef
    );

    const queryData = useLazyLoadQuery<EditVCSProviderQuery>(graphql`
        query EditVCSProviderQuery($id: String!) {
            node(id: $id) {
                ... on VCSProvider {
                    name
                    type
                    description
                    autoCreateWebhooks
                    url
                }
            }
        }
    `, { id: vcsProviderId });

    const [commit, isInFlight] = useMutation<EditVCSProviderMutation>(graphql`
        mutation EditVCSProviderMutation($input: UpdateVCSProviderInput!) {
            updateVCSProvider(input: $input) {
                vcsProvider{
                    id
                    description
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
    const [formData, setFormData] = useState<any>(queryData.node ? {
        type: queryData.node.type,
        name: queryData.node.name,
        description: queryData.node.description,
        autoCreateWebhooks: queryData.node.autoCreateWebhooks,
        url: queryData.node.url
    } : null);

    const onUpdate = () => {
        if (formData) {
            commit({
                variables: {
                    input: {
                        id: vcsProviderId,
                        description: formData.description,
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
                    } else {
                        navigate(`../${data.updateVCSProvider.vcsProvider.id}`);
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

    return formData ? (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={group.fullPath}
                childRoutes={[
                    { title: "vcs providers", path: 'vcs_providers' },
                    { title: formData.name, path: vcsProviderId },
                    { title: "edit", path: 'edit' },
                ]}
            />
            <Typography variant="h5">Edit VCS Provider</Typography>
            <VCSProviderForm
                editMode
                data={formData}
                onChange={(data: FormData) => setFormData(data)}
                error={error}
            />
            <Box marginTop={2}>
                <LoadingButton
                    loading={isInFlight}
                    variant="outlined"
                    color="primary"
                    sx={{ marginRight: 2 }}
                    onClick={onUpdate}>
                    Update VCS Provider
                </LoadingButton>
                <Button component={RouterLink} color="inherit" to={-1 as any}>Cancel</Button>
            </Box>
        </Box>
    ) : <Box>VCS Provider Not found</Box>;
}

export default EditVCSProvider;
