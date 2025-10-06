import LoadingButton from '@mui/lab/LoadingButton';
import { Box, Button, Divider, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useState } from 'react';
import { useFragment, useLazyLoadQuery, useMutation } from 'react-relay/hooks';
import { Link as RouterLink, useNavigate, useParams } from 'react-router-dom';
import { MutationError } from '../../common/error';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import RunnerForm, { FormData } from '../../runners/RunnerForm';
import { EditGroupRunnerFragment_group$key } from './__generated__/EditGroupRunnerFragment_group.graphql';
import { EditGroupRunnerMutation } from './__generated__/EditGroupRunnerMutation.graphql';
import { EditGroupRunnerQuery } from './__generated__/EditGroupRunnerQuery.graphql';

interface Props {
    fragmentRef: EditGroupRunnerFragment_group$key;
}

function EditGroupRunner({ fragmentRef }: Props) {
    const runnerId = useParams().runnerId as string;
    const navigate = useNavigate();

    const group = useFragment<EditGroupRunnerFragment_group$key>(
        graphql`
            fragment EditGroupRunnerFragment_group on Group
            {
                id
                fullPath
            }
        `, fragmentRef
    );

    const queryData = useLazyLoadQuery<EditGroupRunnerQuery>(graphql`
        query EditGroupRunnerQuery($id: String!) {
            node(id: $id) {
                ... on Runner {
                    id
                    name
                    description
                    disabled
                    tags
                    runUntaggedJobs
                }
            }
        }
    `, { id: runnerId });

    const [commit, isInFlight] = useMutation<EditGroupRunnerMutation>(graphql`
        mutation EditGroupRunnerMutation($input: UpdateRunnerInput!) {
            updateRunner(input: $input) {
                runner {
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

    const runner = queryData.node as any;

    const [error, setError] = useState<MutationError>();
    const [formData, setFormData] = useState<FormData>({
        name: runner.name,
        description: runner.description,
        disabled: runner.disabled,
        tags: runner.tags,
        runUntaggedJobs: runner.runUntaggedJobs
    });

    const onUpdate = () => {
        if (formData) {
            commit({
                variables: {
                    input: {
                        id: runner.id,
                        description: formData.description,
                        disabled: formData.disabled,
                        tags: formData.tags,
                        runUntaggedJobs: formData.runUntaggedJobs
                    }
                },
                onCompleted: data => {
                    if (data.updateRunner.problems.length) {
                        setError({
                            severity: 'warning',
                            message: data.updateRunner.problems.map((problem: any) => problem.message).join('; ')
                        });
                    } else if (!data.updateRunner.runner) {
                        setError({
                            severity: 'error',
                            message: "Unexpected error occurred"
                        });
                    } else {
                        navigate(-1);
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
                    { title: "runners", path: 'runners' },
                    { title: formData.name, path: runner.id },
                    { title: "edit", path: 'edit' },
                ]}
            />
            <Typography variant="h5">Edit Runner</Typography>
            <RunnerForm
                editMode
                data={formData}
                onChange={(data: FormData) => setFormData(data)}
                error={error}
            />
            <Divider light />
            <Box mt={2}>
                <LoadingButton
                    loading={isInFlight}
                    variant="outlined"
                    color="primary"
                    sx={{ mr: 2 }}
                    onClick={onUpdate}>
                    Update Runner
                </LoadingButton>
                <Button component={RouterLink} color="inherit" to={-1 as any}>Cancel</Button>
            </Box>
        </Box>
    );
}

export default EditGroupRunner
