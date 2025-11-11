import { useState } from 'react';
import { Alert, Box, Collapse, Typography } from '@mui/material';
import { useFragment, useMutation } from 'react-relay';
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import SettingsToggleButton from '../../common/SettingsToggleButton';
import LabelManager from '../labels/LabelManager';
import { Label } from '../labels/types';
import type { WorkspaceLabelSettingsUpdateMutation } from './__generated__/WorkspaceLabelSettingsUpdateMutation.graphql';

interface Props {
    fragmentRef: any;
}

function WorkspaceLabelSettings(props: Props) {
    const { enqueueSnackbar } = useSnackbar();
    const [showSettings, setShowSettings] = useState<boolean>(false);
    const [error, setError] = useState<{ severity: 'error' | 'warning' | 'info' | 'success'; message: string }>();

    const [updateCommit] = useMutation<WorkspaceLabelSettingsUpdateMutation>(graphql`
        mutation WorkspaceLabelSettingsUpdateMutation($input: UpdateWorkspaceInput!) {
            updateWorkspace(input: $input) {
                workspace {
                    id
                    fullPath
                    labels {
                        key
                        value
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

    const data = useFragment(
        graphql`
        fragment WorkspaceLabelSettingsFragment_workspace on Workspace
        {
            id
            fullPath
            description
            labels {
                key
                value
            }
        }
    `, props.fragmentRef
    );

    const handleSaveLabels = async (labels: Label[]): Promise<void> => {
        return new Promise((resolve, reject) => {
            updateCommit({
                variables: {
                    input: {
                        workspacePath: data.fullPath,
                        description: data.description,
                        labels: labels.map(label => ({
                            key: label.key,
                            value: label.value
                        }))
                    }
                },
                onCompleted: (response) => {
                    if (response?.updateWorkspace?.problems?.length > 0) {
                        const errorMessage = response.updateWorkspace.problems
                            .map((problem: any) => problem.message)
                            .join('; ');
                        setError({
                            severity: 'warning',
                            message: errorMessage
                        });
                        reject(new Error(errorMessage));
                    } else {
                        enqueueSnackbar('Labels updated successfully', { variant: 'success' });
                        setError(undefined);
                        resolve();
                    }
                },
                onError: (err) => {
                    console.error('Workspace labels update error:', err);
                    const errorMessage = err?.message || 'Failed to update labels';
                    setError({
                        severity: 'error',
                        message: errorMessage
                    });
                    reject(new Error(errorMessage));
                }
            });
        });
    };

    // Convert the GraphQL labels to our Label interface
    const currentLabels: Label[] = data.labels?.map((label: any) => ({
        key: label.key,
        value: label.value
    })) || [];

    return (
        <Box>
            {error && <Alert sx={{ mb: 2 }} severity={error.severity}>
                {error.message}
            </Alert>}
            <SettingsToggleButton
                title="Labels"
                showSettings={showSettings}
                onToggle={() => setShowSettings(!showSettings)}
            />
            <Collapse
                in={showSettings}
                timeout="auto"
                unmountOnExit
            >
                <Box sx={{ mt: 2 }}>
                    <Typography variant="body2" color="textSecondary" sx={{ mb: 2 }}>
                        Labels help organize and categorize workspaces. Use them to group workspaces by environment, team, project, or any other criteria.
                    </Typography>

                    <LabelManager
                        labels={currentLabels}
                        onSave={handleSaveLabels}
                        title="Manage Workspace Labels"
                        description="Add, edit, or remove labels for this workspace. Labels are key-value pairs that help with organization and filtering."
                    />
                </Box>
            </Collapse>
        </Box>
    );
}

export default WorkspaceLabelSettings;
