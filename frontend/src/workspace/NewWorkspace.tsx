import React, { useState } from 'react';
import { useMutation } from 'react-relay/hooks';
import graphql from 'babel-plugin-relay/macro';
import { MutationError } from '../common/error';
import { Box, Button, Divider, Typography } from '@mui/material'
import { LoadingButton } from '@mui/lab';
import { useNavigate, useSearchParams } from 'react-router-dom';
import NamespaceBreadcrumbs from '../namespace/NamespaceBreadcrumbs';
import WorkspaceForm, { FormData } from './WorkspaceForm';
import { NewWorkspaceMutation } from './__generated__/NewWorkspaceMutation.graphql';
import { GetConnections } from '../groups/WorkspaceList';

function NewWorkspace(){
    const navigate = useNavigate();
    const [searchParams] = useSearchParams();
    const parentGroupPath: any = searchParams.get('parent');
    const [error, setError] = React.useState<MutationError>()
    const [formData, setFormData] = useState<FormData>({
        name: '',
        description: ''
    });

    const [commit, isInFlight] = useMutation<NewWorkspaceMutation>(graphql`
        mutation NewWorkspaceMutation($input: CreateWorkspaceInput!, $connections: [ID!]!) {
            createWorkspace(input: $input) {
                workspace @prependNode(connections: $connections, edgeTypeName: "WorkspaceEdge") {
                    id
                    name
                    fullPath
                }
                problems {
                    message
                    field
                    type
                }
            }
        }`
    );

    const onCreate = () => {
        commit({
            variables: {
                input: {
                    name: formData.name,
                    description: formData.description,
                    groupPath: parentGroupPath
                },
                connections: GetConnections(parentGroupPath)
            },
            onCompleted: data => {
                if (data.createWorkspace.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.createWorkspace.problems.map(problem => problem.message).join('; ')
                    });
                } else if (!data.createWorkspace.workspace) {
                    setError({
                        severity: 'error',
                        message: "Unexpected error occurred"
                    });
                } else {
                    navigate(`../groups/${data.createWorkspace.workspace?.fullPath}`);
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

    return(
        <Box maxWidth={1200} margin="auto" padding={2}>
            <NamespaceBreadcrumbs
                namespacePath={parentGroupPath}
                childRoutes={[{
                    title: 'new', path: `/workspaces/-/new?parent=${parentGroupPath}`
                }]}/>
            <Typography sx={{ paddingBottom: 2}}variant="h5">New Workspace</Typography>
            <WorkspaceForm
                data={formData}
                onChange={(data: FormData) => setFormData(data)}
                error={error}
            />
            <Divider light />
            <Box marginTop={2}>
                <LoadingButton
                    loading={isInFlight}
                    disabled={!formData.name}
                    variant="outlined"
                    color="primary"
                    sx={{ marginRight: 2 }}
                    onClick={onCreate}
                    >
                    Create Workspace
                </LoadingButton>
                <Button color="inherit" onClick={()=>(navigate(parentGroupPath ? `../groups/${parentGroupPath}` : '..'))}>Cancel</Button>
            </Box>
        </Box>
    )

}

export default NewWorkspace
