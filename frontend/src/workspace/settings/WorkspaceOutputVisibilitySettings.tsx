import { useMemo, useState } from 'react';
import { Box, Button } from '@mui/material';
import { MutationError } from '../../common/error';
import { useFragment, useMutation } from 'react-relay/hooks';
import { useSnackbar } from 'notistack';
import SettingsSection from '../../common/SettingsSection';
import graphql from 'babel-plugin-relay/macro';
import { WorkspaceOutputVisibilitySettingsFragment_workspace$key } from './__generated__/WorkspaceOutputVisibilitySettingsFragment_workspace.graphql';
import { WorkspaceOutputVisibilitySettingsMutation } from './__generated__/WorkspaceOutputVisibilitySettingsMutation.graphql';
import OutputVisibilitySettingsForm, { FormData, toKnownVisibility } from '../../namespace/outputvisibility/OutputVisibilitySettingsForm';

interface Props {
    fragmentRef: WorkspaceOutputVisibilitySettingsFragment_workspace$key;
}

function WorkspaceOutputVisibilitySettings({ fragmentRef }: Props) {
    const { enqueueSnackbar } = useSnackbar();
    const [error, setError] = useState<MutationError>();

    const data = useFragment<WorkspaceOutputVisibilitySettingsFragment_workspace$key>(
        graphql`
        fragment WorkspaceOutputVisibilitySettingsFragment_workspace on Workspace {
            fullPath
            outputVisibility {
                inherited
                value
                ...OutputVisibilitySettingsFormFragment_outputVisibility
            }
        }
        `, fragmentRef
    );

    const [commit, isInFlight] = useMutation<WorkspaceOutputVisibilitySettingsMutation>(graphql`
        mutation WorkspaceOutputVisibilitySettingsMutation($input: UpdateWorkspaceInput!) {
            updateWorkspace(input: $input) {
                workspace {
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

    const noChanges = useMemo(() => {
        return data.outputVisibility?.value === formData?.visibility && data.outputVisibility?.inherited === formData?.inherit;
    }, [data.outputVisibility, formData]);

    const onUpdate = () => {
        commit({
            variables: {
                input: {
                    workspacePath: data.fullPath,
                    outputVisibility: {
                        inherit: formData.inherit,
                        visibility: formData.inherit ? null : formData.visibility
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
            <SettingsSection section="output-visibility" title="Output Visibility Settings">
                <OutputVisibilitySettingsForm
                    formData={formData}
                    onChange={(data) => setFormData(data)}
                    error={error}
                    fragmentRef={data.outputVisibility}
                />
                <Box>
                    <Button
                        sx={{ mt: 2 }}
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
            </SettingsSection>
        </Box>
    );
}

export default WorkspaceOutputVisibilitySettings;
