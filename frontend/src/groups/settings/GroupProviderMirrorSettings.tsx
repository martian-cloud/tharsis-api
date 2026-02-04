import { useMemo, useState } from 'react';
import { Box, Collapse } from '@mui/material';
import LoadingButton from '@mui/lab/LoadingButton';
import { MutationError } from '../../common/error';
import { useFragment, useMutation } from 'react-relay/hooks';
import { useSnackbar } from 'notistack';
import SettingsToggleButton from '../../common/SettingsToggleButton';
import graphql from 'babel-plugin-relay/macro';
import { GroupProviderMirrorSettingsFragment_group$key } from './__generated__/GroupProviderMirrorSettingsFragment_group.graphql';
import { GroupProviderMirrorSettingsMutation } from './__generated__/GroupProviderMirrorSettingsMutation.graphql';
import ProviderMirrorSettingsForm, { FormData } from '../../providermirror/ProviderMirrorSettingsForm';

interface Props {
    fragmentRef: GroupProviderMirrorSettingsFragment_group$key;
}

function GroupProviderMirrorSettings({ fragmentRef }: Props) {
    const { enqueueSnackbar } = useSnackbar();
    const [showSettings, setShowSettings] = useState<boolean>(false);
    const [error, setError] = useState<MutationError>();

    const data = useFragment<GroupProviderMirrorSettingsFragment_group$key>(
        graphql`
        fragment GroupProviderMirrorSettingsFragment_group on Group {
            fullPath
            providerMirrorEnabled {
                inherited
                value
                ...ProviderMirrorSettingsFormFragment_providerMirrorEnabled
            }
        }
        `, fragmentRef
    );

    const [commit, isInFlight] = useMutation<GroupProviderMirrorSettingsMutation>(graphql`
        mutation GroupProviderMirrorSettingsMutation($input: UpdateGroupInput!) {
            updateGroup(input: $input) {
                group {
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

    const [formData, setFormData] = useState<FormData>({
        inherit: data.providerMirrorEnabled.inherited,
        enabled: data.providerMirrorEnabled.value
    });

    const showInheritOption = useMemo(() => !data.fullPath.includes('/'), [data.fullPath]);
    const noChanges = useMemo(() => {
        return data.providerMirrorEnabled?.value === formData?.enabled && data.providerMirrorEnabled?.inherited === formData?.inherit;
    }, [data.providerMirrorEnabled, formData]);

    const onUpdate = () => {
        commit({
            variables: {
                input: {
                    groupPath: data.fullPath,
                    providerMirrorEnabled: {
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
                    showInheritOption={showInheritOption}
                    error={error}
                    fragmentRef={data.providerMirrorEnabled}
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

export default GroupProviderMirrorSettings;
