import { useMemo, useState } from 'react';
import { Box, Collapse } from '@mui/material';
import { LoadingButton } from '@mui/lab';
import graphql from 'babel-plugin-relay/macro';
import { useFragment, useMutation } from 'react-relay/hooks';
import { MutationError } from '../../common/error';
import { useSnackbar } from 'notistack';
import RunnerSettingsForm, { FormData } from '../../runnertags/RunnerSettingsForm';
import SettingsToggleButton from '../../common/SettingsToggleButton';
import { GroupRunnerSettingsFragment_group$key } from './__generated__/GroupRunnerSettingsFragment_group.graphql';
import { GroupRunnerSettingsMutation } from './__generated__/GroupRunnerSettingsMutation.graphql';

interface Props {
    fragmentRef: GroupRunnerSettingsFragment_group$key
}

function GroupRunnerSettings({ fragmentRef }: Props) {
    const { enqueueSnackbar } = useSnackbar();
    const [showSettings, setShowSettings] = useState<boolean>(false);

    const data = useFragment<GroupRunnerSettingsFragment_group$key>(
        graphql`
        fragment GroupRunnerSettingsFragment_group on Group
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

    const [commit, isInFlight] = useMutation<GroupRunnerSettingsMutation>(graphql`
        mutation GroupRunnerSettingsMutation($input: UpdateGroupInput!) {
            updateGroup(input: $input){
                group {
                    id
                    runnerTags {
                        inherited
                        value
                    }
                    ...GroupRunnerSettingsFragment_group
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
            tags: data.runnerTags.inherited ? [] : data.runnerTags.value
        });

    const showInheritOption = useMemo(() => !data.fullPath.includes('/'), [data.fullPath]);
    const noChanges = useMemo(() => {
        const tagsMatch = data.runnerTags?.value.join(',') === formData?.tags.join(',');
        const inheritMatch = data.runnerTags?.inherited === formData?.inherit;

        return showInheritOption ? tagsMatch : tagsMatch && inheritMatch;
    }, [data.runnerTags, formData, showInheritOption]);

    const onUpdate = () => {
        if (formData) {
            commit({
                variables: {
                    input: {
                        groupPath: data.fullPath,
                        runnerTags: {
                            inherit: formData.inherit,
                            tags: formData.inherit ? null : formData.tags
                        }
                    }
                },
                onCompleted: data => {
                    if (data.updateGroup.problems.length) {
                        setError({
                            severity: 'warning',
                            message: data.updateGroup.problems.map(problem => problem.message).join('; ')
                        });
                    } else if (!data.updateGroup.group) {
                        setError({
                            severity: 'error',
                            message: "Unexpected error occurred"
                        });
                    } else {
                        enqueueSnackbar('Runner settings updated', { variant: 'success' });
                        setFormData({
                            inherit: data.updateGroup.group.runnerTags?.inherited ?? false,
                            tags: data.updateGroup.group.runnerTags?.value ?? []
                        });
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
                        showInheritOption={showInheritOption}
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

export default GroupRunnerSettings;
