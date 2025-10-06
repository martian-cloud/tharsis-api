import React, { useState } from 'react'
import { Alert, Box, Collapse } from '@mui/material'
import LoadingButton from '@mui/lab/LoadingButton';
import { MutationError } from '../../common/error';
import MaxJobDurationSetting from './MaxJobDurationSetting'
import TerraformCLIVersionSetting from './TerraformCLIVersionSetting'
import PreventDestroyRunSetting from './PreventDestroyRunSetting'
import { useFragment, useMutation } from 'react-relay'
import { useSnackbar } from 'notistack';
import SettingsToggleButton from '../../common/SettingsToggleButton';
import graphql from 'babel-plugin-relay/macro'
import { WorkspaceRunSettingsFragment_workspace$key } from './__generated__/WorkspaceRunSettingsFragment_workspace.graphql'
import { WorkspaceRunSettingsUpdateMutation } from './__generated__/WorkspaceRunSettingsUpdateMutation.graphql'

interface Props {
    fragmentRef: WorkspaceRunSettingsFragment_workspace$key
}

interface RunSettings {
    maxJobDuration: any
    terraformVersion: string
    preventDestroyPlan: boolean
}

function WorkspaceRunSettings(props: Props) {
    const { enqueueSnackbar } = useSnackbar();
    const [showSettings, setShowSettings] = useState<boolean>(false);

    const data = useFragment(
        graphql`
        fragment WorkspaceRunSettingsFragment_workspace on Workspace
        {
            name
            description
            fullPath
            maxJobDuration
            terraformVersion
            preventDestroyPlan
            ...MaxJobDurationSettingFragment_workspace
        }
    `, props.fragmentRef
    )

    const [error, setError] = useState<MutationError>()
    const [disableSave, setDisableSave] = useState<boolean>(true)
    const [runSettings, setRunSettings] = useState<RunSettings>({
        maxJobDuration: data.maxJobDuration,
        terraformVersion: data.terraformVersion,
        preventDestroyPlan: data.preventDestroyPlan
    })

    const [commit, isInFlight] = useMutation<WorkspaceRunSettingsUpdateMutation>(
        graphql`
        mutation WorkspaceRunSettingsUpdateMutation($input:UpdateWorkspaceInput!) {
            updateWorkspace(input: $input) {
                workspace {
                    fullPath
                    maxJobDuration
                    terraformVersion
                    preventDestroyPlan
                }
                problems {
                    message
                    field
                    type
                }
            }
        }`
    )

    const onUpdate = () => {
        if (isNaN(parseInt(runSettings.maxJobDuration))) {
            setError({
                severity: 'warning',
                message: 'Please reenter maximum job duration'
            })
        } else {
            commit({
                variables: {
                    input: {
                        workspacePath: data.fullPath,
                        maxJobDuration: parseInt(runSettings.maxJobDuration),
                        terraformVersion: runSettings.terraformVersion,
                        preventDestroyPlan: runSettings.preventDestroyPlan
                    }
                },
                onCompleted: data => {
                    if (data.updateWorkspace.problems.length) {
                        setError({
                            severity: 'warning',
                            message: data.updateWorkspace.problems.map(problem => problem.message).join('; ')
                        })
                    } else if (!data.updateWorkspace.workspace) {
                        setError({
                            severity: 'error',
                            message: "Unexpected error occurred"
                        })
                    } else {
                        enqueueSnackbar('Settings updated', { variant: 'success' });
                        setDisableSave(true)
                    }
                },
                onError: error => {
                    setError({
                        severity: 'error',
                        message: `Unexpected error occurred: ${error.message}`
                    })
                }
            })
        }
    }

    const onChange = (data: RunSettings) => {
        setRunSettings(data)
        if (disableSave) {
            setDisableSave(false)
        }
        setError(undefined)
    }

    return (
        <Box>
            {error && <Alert sx={{ mb: 2 }} severity={error.severity}>
                {error.message}
            </Alert>}
            <SettingsToggleButton
                title="Run Settings"
                showSettings={showSettings}
                onToggle={() => setShowSettings(!showSettings)}
            />
            <Collapse
                in={showSettings}
                timeout="auto"
            >
                <Box>
                    <MaxJobDurationSetting
                        fragmentRef={data}
                        data={runSettings.maxJobDuration}
                        onChange={(event: any) => onChange({ ...runSettings, maxJobDuration: event.target.value })
                        }
                    />
                    <TerraformCLIVersionSetting
                        data={runSettings.terraformVersion}
                        onChange={(event: any) => {
                            onChange({ ...runSettings, terraformVersion: event.target.value })
                        }}
                    />
                    <PreventDestroyRunSetting
                        data={runSettings.preventDestroyPlan}
                        onChange={(event: any) => {
                            onChange({ ...runSettings, preventDestroyPlan: event.target.checked })
                        }}
                    />
                    <Box>
                        <LoadingButton
                            size="small"
                            disabled={disableSave}
                            loading={isInFlight}
                            variant="outlined"
                            color="primary"
                            onClick={onUpdate}
                        >
                            Save changes
                        </LoadingButton>
                    </Box>
                </Box>
            </Collapse>
        </Box>
    );
}

export default WorkspaceRunSettings;
