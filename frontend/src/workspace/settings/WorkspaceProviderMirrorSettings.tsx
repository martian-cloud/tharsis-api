import { useMemo, useState } from 'react';
import { Box, Collapse } from '@mui/material';
import LoadingButton from '@mui/lab/LoadingButton';
import { MutationError } from '../../common/error';
import { useFragment, useMutation } from 'react-relay/hooks';
import { useSnackbar } from 'notistack';
import SettingsToggleButton from '../../common/SettingsToggleButton';
import graphql from 'babel-plugin-relay/macro';
import { WorkspaceProviderMirrorSettingsFragment_workspace$key } from './__generated__/WorkspaceProviderMirrorSettingsFragment_workspace.graphql';
import { WorkspaceProviderMirrorSettingsMutation } from './__generated__/WorkspaceProviderMirrorSettingsMutation.graphql';
import ProviderMirrorSettingsForm, { FormData } from '../../providermirror/ProviderMirrorSettingsForm';

interface Props {
    fragmentRef: WorkspaceProviderMirrorSettingsFragment_workspace$key;
}

function WorkspaceProviderMirrorSettings({ fragmentRef }: Props) {
    const { enqueueSnackbar } = useSnackbar();
    const [showSettings, setShowSettings] = useState<boolean>(false);
    const [error, setError] = useState<MutationError>();

    const data = useFragment<WorkspaceProviderMirrorSettingsFragment_workspace$key>(
        graphql`
        fragment WorkspaceProviderMirrorSettingsFragment_workspace on Workspace {
            fullPath
            providerMirrorEnabled {
                inherited
                value
                ...ProviderMirrorSettingsFormFragment_providerMirrorEnabled
            }
        }
        `, fragmentRef
    );

    const [commit, isInFlight] = useMutation<WorkspaceProviderMirrorSettingsMutation>(graphql`
        mutation WorkspaceProviderMirrorSettingsMutation($input: UpdateWorkspaceInput!) {
            updateWorkspace(input: $input) {
                workspace {
                    providerMirrorEnabled {
                        ...ProviderMirrorSettingsFormFragment_providerMirrorEnabled
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
            inherit: data.providerMirrorEnabled.inherited,
            enabled: data.providerMirrorEnabled.value
        });

    const noChanges = useMemo(() => {
        return data.providerMirrorEnabled?.value === formData?.enabled && data.providerMirrorEnabled?.inherited === formData?.inherit;
    }, [data.providerMirrorEnabled, formData]);

    const onUpdate = () => {
        commit({
            variables: {
                input: {
                    workspacePath: data.fullPath,
                    providerMirrorEnabled: {
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
                    enqueueSnackbar('Provider mirror settings updated', { variant: 'success' });
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
                title="Provider Mirror Settings"
                showSettings={showSettings}
                onToggle={() => setShowSettings(!showSettings)}
            />
            <Collapse
                in={showSettings}
                timeout="auto"
                unmountOnExit
            >
                <ProviderMirrorSettingsForm
                    formData={formData}
                    onChange={(data) => setFormData(data)}
                    error={error}
                    fragmentRef={data.providerMirrorEnabled}
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

export default WorkspaceProviderMirrorSettings;
