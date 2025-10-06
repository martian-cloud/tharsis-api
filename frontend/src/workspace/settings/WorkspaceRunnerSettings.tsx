import { useMemo, useState } from 'react';
import { Box, Collapse } from '@mui/material';
import { LoadingButton } from '@mui/lab';
import graphql from 'babel-plugin-relay/macro';
import { useFragment, useMutation } from 'react-relay/hooks';
import { MutationError } from '../../common/error';
import { useSnackbar } from 'notistack';
import RunnerSettingsForm from '../../runnertags/RunnerSettingsForm';
import SettingsToggleButton from '../../common/SettingsToggleButton';
import { FormData } from '../../runnertags/RunnerSettingsForm';
import { WorkspaceRunnerSettingsFragment_workspace$key } from './__generated__/WorkspaceRunnerSettingsFragment_workspace.graphql';
import { WorkspaceRunnerSettingsMutation } from './__generated__/WorkspaceRunnerSettingsMutation.graphql';

interface Props {
    fragmentRef: WorkspaceRunnerSettingsFragment_workspace$key
}

function WorkspaceRunnerSettings({ fragmentRef }: Props) {
    const { enqueueSnackbar } = useSnackbar();
    const [showSettings, setShowSettings] = useState<boolean>(false);

    const data = useFragment<WorkspaceRunnerSettingsFragment_workspace$key>(
        graphql`
        fragment WorkspaceRunnerSettingsFragment_workspace on Workspace
        {
            fullPath
            runnerTags {
                inherited
                namespacePath
                value
                ...RunnerSettingsForm_runnerTags
            }
        }
        `, fragmentRef
    );

    const [commit, isInFlight] = useMutation<WorkspaceRunnerSettingsMutation>(graphql`
        mutation WorkspaceRunnerSettingsMutation($input: UpdateWorkspaceInput!) {
            updateWorkspace(input: $input){
                workspace{
                    id
                    runnerTags {
                        inherited
                        value
                    }
                    ...WorkspaceRunnerSettingsFragment_workspace
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const [error, setError] = useState<MutationError>();
    const [formData, setFormData] = useState<FormData>(
        {
            inherit: data.runnerTags.inherited,
            tags: data.runnerTags.value
        });

    const noChanges = useMemo(() => {
        return data.runnerTags?.value.join(',') === formData?.tags.join(',') && data.runnerTags?.inherited === formData?.inherit;
     }, [data.runnerTags, formData]);

    const onUpdate = () => {
        if (formData) {
            commit({
                variables: {
                    input: {
                        workspacePath: data.fullPath,
                        runnerTags: {
                            inherit: formData.inherit,
                            tags: formData.inherit ? null : formData.tags
                        }
                    }
                },
                onCompleted: data => {
                    if (data.updateWorkspace.problems.length) {
                        setError({
                            severity: 'warning',
                            message: data.updateWorkspace.problems.map(problem => problem.message).join('; ')
                        });
                    } else if (!data.updateWorkspace.workspace) {
                        setError({
                            severity: 'error',
                            message: "Unexpected error occurred"
                        });
                    } else {
                        enqueueSnackbar('Runner settings updated', { variant: 'success' });
                        setFormData({
                            inherit: data.updateWorkspace.workspace.runnerTags?.inherited ?? false,
                            tags: data.updateWorkspace.workspace.runnerTags?.value ?? []
                        })
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

    const handleInputChange = (runnerTags: FormData) => {
        setFormData(runnerTags);
        setError(undefined);
    };

    return formData ? (
        <Box>
            <SettingsToggleButton
                title="Runner Settings"
                showSettings={showSettings}
                onToggle={() => setShowSettings(!showSettings)}
            />
            <Collapse
                in={showSettings}
                timeout="auto"
                unmountOnExit
            >
                {data.runnerTags &&
                    <RunnerSettingsForm
                        onChange={handleInputChange}
                        formData={formData}
                        fragmentRef={data.runnerTags}
                        error={error}
                    />}
                <Box marginTop={3}>
                    <LoadingButton
                        size="small"
                        loading={isInFlight}
                        disabled={noChanges}
                        variant="outlined"
                        color="primary"
                        sx={{ marginRight: 2 }}
                        onClick={onUpdate}>
                        Save Changes
                    </LoadingButton>
                </Box>
            </Collapse>
        </Box>
    ) : <Box>Not found</Box>;
}

export default WorkspaceRunnerSettings;
