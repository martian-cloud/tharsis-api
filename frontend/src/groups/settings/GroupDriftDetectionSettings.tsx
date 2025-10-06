import { useMemo, useState } from 'react';
import { Box, Collapse } from '@mui/material';
import LoadingButton from '@mui/lab/LoadingButton';
import { MutationError } from '../../common/error';
import { useFragment, useMutation } from 'react-relay/hooks';
import { useSnackbar } from 'notistack';
import SettingsToggleButton from '../../common/SettingsToggleButton';
import graphql from 'babel-plugin-relay/macro';
import { GroupDriftDetectionSettingsFragment_group$key } from './__generated__/GroupDriftDetectionSettingsFragment_group.graphql';
import { GroupDriftDetectionSettingsMutation } from './__generated__/GroupDriftDetectionSettingsMutation.graphql';
import DriftDetectionSettingsForm, { FormData } from '../../driftdetection/DriftDetectionSettingsForm';

interface Props {
    fragmentRef: GroupDriftDetectionSettingsFragment_group$key;
}

function GroupDriftDetectionSettings({ fragmentRef }: Props) {
    const { enqueueSnackbar } = useSnackbar();
    const [showSettings, setShowSettings] = useState<boolean>(false);
    const [error, setError] = useState<MutationError>();

    const data = useFragment<GroupDriftDetectionSettingsFragment_group$key>(
        graphql`
        fragment GroupDriftDetectionSettingsFragment_group on Group {
            fullPath
            driftDetectionEnabled {
                inherited
                value
                ...DriftDetectionSettingsFormFragment_driftDetectionEnabled
            }
        }
        `, fragmentRef
    );

    const [commit, isInFlight] = useMutation<GroupDriftDetectionSettingsMutation>(graphql`
        mutation GroupDriftDetectionSettingsMutation($input: UpdateGroupInput!) {
            updateGroup(input: $input) {
                group {
                    driftDetectionEnabled {
                        ...DriftDetectionSettingsFormFragment_driftDetectionEnabled
                    }
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const [formData, setFormData] = useState<FormData>({
        inherit: data.driftDetectionEnabled.inherited,
        enabled: data.driftDetectionEnabled.value
    });

    const showInheritOption = useMemo(() => !data.fullPath.includes('/'), [data.fullPath]);
    const noChanges = useMemo(() => {
        return data.driftDetectionEnabled?.value === formData?.enabled && data.driftDetectionEnabled?.inherited === formData?.inherit;
    }, [data.driftDetectionEnabled, formData]);

    const onUpdate = () => {
        commit({
            variables: {
                input: {
                    groupPath: data.fullPath,
                    driftDetectionEnabled: {
                        inherit: formData.inherit,
                        enabled: formData.inherit ? null : formData.enabled
                    }
                }
            },
            onCompleted: data => {
                if (data.updateGroup.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.updateGroup.problems.map((problem: { message: any }) => problem.message).join('; ')
                    });
                } else if (!data.updateGroup.group) {
                    setError({
                        severity: 'error',
                        message: "Unexpected error occurred"
                    });
                } else {
                    enqueueSnackbar('Drift detection settings updated', { variant: 'success' });
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
            <SettingsToggleButton
                title="Drift Detection Settings"
                showSettings={showSettings}
                onToggle={() => setShowSettings(!showSettings)}
            />
            <Collapse
                in={showSettings}
                timeout="auto"
                unmountOnExit
            >
                <DriftDetectionSettingsForm
                    formData={formData}
                    onChange={(data) => setFormData(data)}
                    showInheritOption={showInheritOption}
                    error={error}
                    fragmentRef={data.driftDetectionEnabled}
                />
                <Box>
                    <LoadingButton
                        sx={{ mt: 4 }}
                        size="small"
                        disabled={noChanges}
                        loading={isInFlight}
                        variant="outlined"
                        color="primary"
                        onClick={onUpdate}
                    >
                        Save changes
                    </LoadingButton>
                </Box>
            </Collapse>
        </Box>
    );
}

export default GroupDriftDetectionSettings;
