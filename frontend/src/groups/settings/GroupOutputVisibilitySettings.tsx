import { useMemo, useState } from 'react';
import { Box, Button, Collapse } from '@mui/material';
import { MutationError } from '../../common/error';
import { useFragment, useMutation } from 'react-relay/hooks';
import { useSnackbar } from 'notistack';
import SettingsToggleButton from '../../common/SettingsToggleButton';
import graphql from 'babel-plugin-relay/macro';
import { GroupOutputVisibilitySettingsFragment_group$key } from './__generated__/GroupOutputVisibilitySettingsFragment_group.graphql';
import { GroupOutputVisibilitySettingsMutation } from './__generated__/GroupOutputVisibilitySettingsMutation.graphql';
import OutputVisibilitySettingsForm, { FormData, toKnownVisibility } from '../../namespace/outputvisibility/OutputVisibilitySettingsForm';

interface Props {
    fragmentRef: GroupOutputVisibilitySettingsFragment_group$key;
}

function GroupOutputVisibilitySettings({ fragmentRef }: Props) {
    const { enqueueSnackbar } = useSnackbar();
    const [showSettings, setShowSettings] = useState<boolean>(false);
    const [error, setError] = useState<MutationError>();

    const data = useFragment<GroupOutputVisibilitySettingsFragment_group$key>(
        graphql`
        fragment GroupOutputVisibilitySettingsFragment_group on Group {
            fullPath
            outputVisibility {
                inherited
                value
                ...OutputVisibilitySettingsFormFragment_outputVisibility
            }
        }
        `, fragmentRef
    );

    const [commit, isInFlight] = useMutation<GroupOutputVisibilitySettingsMutation>(graphql`
        mutation GroupOutputVisibilitySettingsMutation($input: UpdateGroupInput!) {
            updateGroup(input: $input) {
                group {
                    outputVisibility {
                        ...OutputVisibilitySettingsFormFragment_outputVisibility
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
        inherit: data.outputVisibility.inherited,
        visibility: toKnownVisibility(data.outputVisibility.value)
    });

    const isRootGroup = useMemo(() => !data.fullPath.includes('/'), [data.fullPath]);
    const noChanges = useMemo(() => {
        return data.outputVisibility?.value === formData?.visibility && data.outputVisibility?.inherited === formData?.inherit;
    }, [data.outputVisibility, formData]);

    const onUpdate = () => {
        commit({
            variables: {
                input: {
                    groupPath: data.fullPath,
                    outputVisibility: {
                        inherit: formData.inherit,
                        visibility: formData.inherit ? null : formData.visibility
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
                    enqueueSnackbar('Output visibility settings updated', { variant: 'success' });
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
                title="Output Visibility Settings"
                showSettings={showSettings}
                onToggle={() => setShowSettings(!showSettings)}
            />
            <Collapse
                in={showSettings}
                timeout="auto"
                unmountOnExit
            >
                <OutputVisibilitySettingsForm
                    formData={formData}
                    onChange={(data) => setFormData(data)}
                    isRootGroup={isRootGroup}
                    error={error}
                    fragmentRef={data.outputVisibility}
                />
                <Box>
                    <Button
                        sx={{ mt: 4 }}
                        size="small"
                        disabled={noChanges}
                        loading={isInFlight}
                        variant="outlined"
                        color="primary"
                        onClick={onUpdate}
                    >
                        Save changes
                    </Button>
                </Box>
            </Collapse>
        </Box>
    );
}

export default GroupOutputVisibilitySettings;
