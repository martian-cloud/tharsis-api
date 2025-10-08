import { useMemo, useState } from 'react';
import { Alert, Box, Collapse } from '@mui/material';
import LoadingButton from '@mui/lab/LoadingButton';
import { MutationError } from '../../common/error';
import LockWorkspaceSetting from './LockWorkspaceSetting';
import { useFragment, useMutation } from 'react-relay'
import { useSnackbar } from 'notistack';
import graphql from 'babel-plugin-relay/macro';
import SettingsToggleButton from '../../common/SettingsToggleButton';
import { WorkspaceStateSettingsFragment_workspace$key } from './__generated__/WorkspaceStateSettingsFragment_workspace.graphql';
import { WorkspaceStateSettingsLockWorkspaceMutation } from './__generated__/WorkspaceStateSettingsLockWorkspaceMutation.graphql';
import { WorkspaceStateSettingsUnlockWorkspaceMutation } from './__generated__/WorkspaceStateSettingsUnlockWorkspaceMutation.graphql';

interface Props {
    fragmentRef: WorkspaceStateSettingsFragment_workspace$key;
}

interface StateSettings {
    locked: boolean;
}

function WorkspaceStateSettings(props: Props) {
    const { enqueueSnackbar } = useSnackbar();

    const data = useFragment(
        graphql`
        fragment WorkspaceStateSettingsFragment_workspace on Workspace
        {
            fullPath
            locked
        }
    `, props.fragmentRef
    )

    const [error, setError] = useState<MutationError>()
    const [disableSave, setDisableSave] = useState<boolean>(true)
    const [stateSettings, setStateSettings] = useState<StateSettings>({
        locked: data.locked,
    })
    const [showSettings, setShowSettings] = useState<boolean>(false);

    const [lockWorkspaceCommit, lockWorkspaceIsInFlight] = useMutation<WorkspaceStateSettingsLockWorkspaceMutation>(
        graphql`
        mutation WorkspaceStateSettingsLockWorkspaceMutation($input:LockWorkspaceInput!) {
            lockWorkspace(input: $input) {
                workspace {
                    fullPath
                    locked
                }
                problems {
                    message
                    field
                    type
                }
            }
        }`
    )

    const [unlockWorkspaceCommit, unlockWorkspaceIsInFlight] = useMutation<WorkspaceStateSettingsUnlockWorkspaceMutation>(
        graphql`
        mutation WorkspaceStateSettingsUnlockWorkspaceMutation($input:UnlockWorkspaceInput!) {
            unlockWorkspace(input: $input) {
                workspace {
                    fullPath
                    locked
                }
                problems {
                    message
                    field
                    type
                }
            }
        }`
    )

    const isInFlight = useMemo(() => lockWorkspaceIsInFlight || unlockWorkspaceIsInFlight, [lockWorkspaceIsInFlight, unlockWorkspaceIsInFlight]);

    const lockWorkspace = () => {
        lockWorkspaceCommit({
            variables: {
                input: {
                    workspacePath: data.fullPath,
                }
            },
            onCompleted: data => {
                if (data.lockWorkspace.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.lockWorkspace.problems.map(problem => problem.message).join('; ')
                    })
                } else if (!data.lockWorkspace.workspace) {
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

    const unlockWorkspace = () => {
        unlockWorkspaceCommit({
            variables: {
                input: {
                    workspacePath: data.fullPath,
                }
            },
            onCompleted: data => {
                if (data.unlockWorkspace.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.unlockWorkspace.problems.map(problem => problem.message).join('; ')
                    })
                } else if (!data.unlockWorkspace.workspace) {
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

    const onUpdate = () => {
        if (stateSettings.locked) {
            lockWorkspace()
            return
        }

        unlockWorkspace()
    }

    const onChange = (data: StateSettings) => {
        setStateSettings(data)
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
                title="State Settings"
                showSettings={showSettings}
                onToggle={() => setShowSettings(!showSettings)}
            />
            <Collapse
                in={showSettings}
                timeout="auto"
                unmountOnExit
            >
                <Box>
                    <LockWorkspaceSetting
                        locked={stateSettings.locked}
                        onChange={(event: any) => {
                            onChange({ ...stateSettings, locked: event.target.checked })
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

export default WorkspaceStateSettings;
