import { useState } from 'react'
import { Alert, Box, Collapse, TextField, Typography } from '@mui/material'
import LoadingButton from '@mui/lab/LoadingButton';
import { MutationError } from '../../common/error';
import { useFragment, useMutation } from 'react-relay';
import graphql from 'babel-plugin-relay/macro'
import { useSnackbar } from 'notistack';
import { GroupGeneralSettingsFragment_group$key } from './__generated__/GroupGeneralSettingsFragment_group.graphql'
import { GroupGeneralSettingsUpdateMutation } from './__generated__/GroupGeneralSettingsUpdateMutation.graphql'
import SettingsToggleButton from '../../common/SettingsToggleButton';

interface Props {
    fragmentRef: GroupGeneralSettingsFragment_group$key
}

function GroupGeneralSettings(props: Props) {
    const { enqueueSnackbar } = useSnackbar();
    const [showSettings, setShowSettings] = useState<boolean>(false);

    const data = useFragment(
        graphql`
        fragment GroupGeneralSettingsFragment_group on Group
        {
            name
            description
            fullPath
        }
    `, props.fragmentRef
    )

    const [error, setError] = useState<MutationError>()
    const [inputForm, setInputForm] = useState<{name: string, description: string}>({
        name: data.name,
        description: data.description
    })

    const [commit, isInFlight] = useMutation<GroupGeneralSettingsUpdateMutation>(
        graphql`
        mutation GroupGeneralSettingsUpdateMutation($input: UpdateGroupInput!) {
            updateGroup(input: $input) {
                group {
                    id
                    ...GroupListItemFragment_group
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `)

const onUpdate = () => {
    commit({
        variables: {
            input: {
                groupPath: data.fullPath,
                description: inputForm.description,
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
                enqueueSnackbar('Settings updated', { variant: 'success' });
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
            {error && <Alert sx={{ mb: 2 }} severity={error.severity}>
                {error.message}
            </Alert>}
            <SettingsToggleButton
                title="General Settings"
                showSettings={showSettings}
                onToggle={() => setShowSettings(!showSettings)}
            />
            <Collapse
                in={showSettings}
                timeout="auto"
                unmountOnExit
            >
                <Box>
                    <Typography mt={2} mb={2} variant="subtitle1" gutterBottom>Details</Typography>
                    <TextField
                        disabled
                        size="small"
                        fullWidth
                        label="Name"
                        value={data.name}
                        onChange={event => setInputForm({ ...inputForm, name: event.target.value })}
                    />
                    <TextField
                        size="small"
                        margin='normal'
                        fullWidth
                        label="Description"
                        value={inputForm.description}
                        onChange={event => setInputForm({ ...inputForm, description: event.target.value })}
                    />
                </Box>
                <Box>
                    <LoadingButton
                        sx={{ mt: 1 }}
                        size="small"
                        disabled={data.description === inputForm.description}
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

export default GroupGeneralSettings
