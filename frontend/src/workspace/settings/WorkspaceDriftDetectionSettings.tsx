import { useMemo, useState } from 'react';
import { Box, Collapse } from '@mui/material';
import LoadingButton from '@mui/lab/LoadingButton';
import { MutationError } from '../../common/error';
import { useFragment, useMutation } from 'react-relay/hooks';
import { useSnackbar } from 'notistack';
import SettingsToggleButton from '../../common/SettingsToggleButton';
import graphql from 'babel-plugin-relay/macro';
import { WorkspaceDriftDetectionSettingsFragment_workspace$key } from './__generated__/WorkspaceDriftDetectionSettingsFragment_workspace.graphql';
import { WorkspaceDriftDetectionSettingsMutation } from './__generated__/WorkspaceDriftDetectionSettingsMutation.graphql';
import DriftDetectionSettingsForm, { FormData } from '../../driftdetection/DriftDetectionSettingsForm';

interface Props {
    fragmentRef: WorkspaceDriftDetectionSettingsFragment_workspace$key;
}

function WorkspaceDriftDetectionSettings({ fragmentRef }: Props) {
    const { enqueueSnackbar } = useSnackbar();
    const [showSettings, setShowSettings] = useState<boolean>(false);
    const [error, setError] = useState<MutationError>();

    const data = useFragment<WorkspaceDriftDetectionSettingsFragment_workspace$key>(
        graphql`
        fragment WorkspaceDriftDetectionSettingsFragment_workspace on Workspace {
            fullPath
            driftDetectionEnabled {
                inherited
                value
                ...DriftDetectionSettingsFormFragment_driftDetectionEnabled
            }
        }
        `, fragmentRef
    );

    const [commit, isInFlight] = useMutation<WorkspaceDriftDetectionSettingsMutation>(graphql`
        mutation WorkspaceDriftDetectionSettingsMutation($input: UpdateWorkspaceInput!) {
            updateWorkspace(input: $input) {
                workspace {
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

    const [formData, setFormData] = useState<FormData>(
        {
            inherit: data.driftDetectionEnabled.inherited,
            enabled: data.driftDetectionEnabled.value
        });

    const noChanges = useMemo(() => {
        return data.driftDetectionEnabled?.value === formData?.enabled && data.driftDetectionEnabled?.inherited === formData?.inherit;
    }, [data.driftDetectionEnabled, formData]);

    const onUpdate = () => {
        commit({
            variables: {
                input: {
                    workspacePath: data.fullPath,
                    driftDetectionEnabled: {
                        inherit: formData.inherit,
                        enabled: formData.inherit ? null : formData.enabled
                    }
                }
            },
            onCompleted: data => {
                if (data.updateWorkspace.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.updateWorkspace.problems.map((problem: { message: any }) => problem.message).join('; ')
                    });
                } else if (!data.updateWorkspace.workspace) {
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
                    error={error}
                    fragmentRef={data.driftDetectionEnabled}
                />
                <Box>
                    <LoadingButton
                        sx={{ mt: 2 }}
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

export default WorkspaceDriftDetectionSettings;
