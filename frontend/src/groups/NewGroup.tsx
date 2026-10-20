import { Breadcrumbs, Button } from '@mui/material';
import Box from '@mui/material/Box';
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
import { PageLayoutProvider } from '../layout/PageLayoutContext';

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
        mutation NewGroupMutation($input: CreateGroupInput!) {
            createGroup(input: $input) {
                group {
                    id
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
                    parentPath: parentGroupPath
                },
            },
            // Namespace lists are cached per view, search and expanded group, so mark the whole store stale and let
            // whichever list mounts next refetch instead of patching each connection
            updater: (store, data) => {
                if (data?.createGroup.group) {
                    store.invalidateStore();
                }
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
        <PageLayoutProvider>
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
                <Button
                    loading={isInFlight}
                    disabled={!formData.name}
                    variant="outlined"
                    color="primary"
                    sx={{ marginRight: 2 }}
                    onClick={onCreate}
                >
                    Create Group
                </Button>
                <Button color="inherit" onClick={() => (navigate(parentGroupPath ? `../groups/${parentGroupPath}` : '..'))}>Cancel</Button>
            </Box>
        </PageLayoutProvider>
    );
}

export default NewGroup;
