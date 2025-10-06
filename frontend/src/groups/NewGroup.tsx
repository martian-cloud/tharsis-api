import LoadingButton from '@mui/lab/LoadingButton';
import { Breadcrumbs } from '@mui/material';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Divider from '@mui/material/Divider';
import Typography from '@mui/material/Typography';
import graphql from 'babel-plugin-relay/macro';
import React, { useState } from 'react';
import { useMutation } from "react-relay/hooks";
import { useNavigate, useSearchParams } from 'react-router-dom';
import { MutationError } from '../common/error';
import NamespaceBreadcrumbs from '../namespace/NamespaceBreadcrumbs';
import Link from '../routes/Link';
import { NewGroupMutation } from './__generated__/NewGroupMutation.graphql';
import GroupForm, { FormData } from './GroupForm';
import { GetConnections } from './GroupList';
import { GetConnections as GetTopLevelConnections } from './tree/GroupTreeContainer';

function NewGroup() {
    const navigate = useNavigate();
    const [searchParams] = useSearchParams();
    const parentGroupPath = searchParams.get('parent');
    const [error, setError] = React.useState<MutationError>();
    const [formData, setFormData] = useState<FormData>({
        name: '',
        description: ''
    });

    const [commit, isInFlight] = useMutation<NewGroupMutation>(graphql`
        mutation NewGroupMutation($input: CreateGroupInput!, $connections: [ID!]!) {
            createGroup(input: $input) {
                group @prependNode(connections: $connections, edgeTypeName: "GroupEdge") {
                    id
                    fullPath
                    ...GroupListItemFragment_group
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
                    parentPath: parentGroupPath
                },
                connections: parentGroupPath ? GetConnections(parentGroupPath) : GetTopLevelConnections()
            },

            onCompleted: data => {
                if (data.createGroup.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.createGroup.problems.map(problem => problem.message).join('; ')
                    });

                } else if (!data.createGroup.group) {
                    setError({
                        severity: 'error',
                        message: "Unexpected error occurred"
                    });
                } else {
                    navigate(`../groups/${data.createGroup.group.fullPath}`);
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
        <Box maxWidth={1200} margin="auto" padding={2}>
            {parentGroupPath ?
                <NamespaceBreadcrumbs
                    namespacePath={parentGroupPath}
                    childRoutes={[{
                        title: 'new', path: `/groups/-/new?parent=${parentGroupPath}`
                    }]}
                />
                :
                <Breadcrumbs aria-label="group breadcrumb" sx={{ marginBottom: 2 }}>
                    <Link color="inherit" to={'/groups'}>
                        groups
                    </Link>
                    <Link color="inherit" to={`/groups/-/new`} >
                        new
                    </Link>
                </Breadcrumbs>
            }
            <Typography sx={{ paddingBottom: 2 }} variant="h5">New Group</Typography>
            <GroupForm
                data={formData}
                onChange={(data: FormData) => setFormData(data)}
                error={error} />
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
                    Create Group
                </LoadingButton>
                <Button color="inherit" onClick={() => (navigate(parentGroupPath ? `../groups/${parentGroupPath}` : '..'))}>Cancel</Button>
            </Box>
        </Box>
    );
}

export default NewGroup;
