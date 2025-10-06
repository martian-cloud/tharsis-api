import LoadingButton from '@mui/lab/LoadingButton';
import { Box, Button, Divider, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useState } from 'react';
import { useFragment, useMutation } from 'react-relay/hooks';
import { Link as RouterLink, useNavigate } from 'react-router-dom';
import { MutationError } from '../../common/error';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import RunnerForm, { FormData } from '../../runners/RunnerForm';
import { GetConnections } from './GroupRunnersList';
import { NewGroupRunnerFragment_group$key } from './__generated__/NewGroupRunnerFragment_group.graphql';
import { NewGroupRunnerMutation } from './__generated__/NewGroupRunnerMutation.graphql';

interface Props {
    fragmentRef: NewGroupRunnerFragment_group$key;
}

function NewGroupRunner({ fragmentRef }: Props) {
    const navigate = useNavigate();

    const group = useFragment<NewGroupRunnerFragment_group$key>(graphql`
        fragment NewGroupRunnerFragment_group on Group
        {
            id
            fullPath
        }
    `, fragmentRef);

    const [commit, isInFlight] = useMutation<NewGroupRunnerMutation>(graphql`
        mutation NewGroupRunnerMutation($input: CreateRunnerInput!, $connections: [ID!]!) {
            createRunner(input: $input) {
                # Use @prependNode to add the node to the connection
                runner  @prependNode(connections: $connections, edgeTypeName: "RunnerEdge") {
                    id
                    ...RunnerListItemFragment_runner
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
        disabled: false,
        tags: [],
        runUntaggedJobs: false
    });

    const onSave = () => {
        commit({
            variables: {
                input: {
                    name: formData.name,
                    description: formData.description,
                    groupPath: group.fullPath,
                    disabled: formData.disabled,
                    runUntaggedJobs: formData.runUntaggedJobs,
                    tags: formData.tags
                },
                connections: GetConnections(group.id)
            },
            onCompleted: data => {
                if (data.createRunner.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.createRunner.problems.map((problem: any) => problem.message).join('; ')
                    });
                } else if (!data.createRunner.runner) {
                    setError({
                        severity: 'error',
                        message: "Unexpected error occurred"
                    });
                } else {
                    navigate(`../${data.createRunner.runner.id}`);
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
                    { title: "runners", path: 'runners' },
                    { title: "new", path: 'new' },
                ]}
            />
            <Typography variant="h5">New Runner</Typography>
            <RunnerForm
                data={formData}
                onChange={(data: FormData) => setFormData(data)}
                error={error}
            />
            <Divider light />
            <Box mt={2}>
                <LoadingButton
                    disabled={!formData.name}
                    loading={isInFlight}
                    variant="outlined"
                    color="primary"
                    sx={{ mr: 2 }}
                    onClick={onSave}
                >
                    Create Runner
                </LoadingButton>
                <Button component={RouterLink} color="inherit" to={-1 as any}>Cancel</Button>
            </Box>
        </Box>
    );
}

export default NewGroupRunner
