import {
    Alert,
    Box,
    FormControlLabel,
    Switch,
    Typography
} from "@mui/material";
import LoadingButton from '@mui/lab/LoadingButton';
import graphql from 'babel-plugin-relay/macro';
import { useState } from "react";
import { useLazyLoadQuery, useMutation } from "react-relay/hooks";
import { useSnackbar } from "notistack";
import { MutationError } from "../../common/error";
import Timestamp from "../../common/Timestamp";
import { MaintenanceSettingsQuery } from "./__generated__/MaintenanceSettingsQuery.graphql";
import { MaintenanceSettingsEnableMutation } from "./__generated__/MaintenanceSettingsEnableMutation.graphql";
import { MaintenanceSettingsDisableMutation } from "./__generated__/MaintenanceSettingsDisableMutation.graphql";

function MaintenanceSettings() {
    const { enqueueSnackbar } = useSnackbar();
    const [error, setError] = useState<MutationError>();

    const data = useLazyLoadQuery<MaintenanceSettingsQuery>(graphql`
        query MaintenanceSettingsQuery {
            maintenanceMode {
                id
                createdBy
                metadata {
                    createdAt
                }
            }
        }
    `, {}, { fetchPolicy: 'network-only' });

    const [currentMaintenanceMode, setCurrentMaintenanceMode] = useState(data.maintenanceMode);
    const [isMaintenanceModeEnabled, setIsMaintenanceModeEnabled] = useState(!!data.maintenanceMode);

    const [enableMaintenanceMode, isEnabling] = useMutation<MaintenanceSettingsEnableMutation>(graphql`
        mutation MaintenanceSettingsEnableMutation($input: EnableMaintenanceModeInput!) {
            enableMaintenanceMode(input: $input) {
                maintenanceMode {
                    id
                    createdBy
                    metadata {
                        createdAt
                    }
                }
                problems {
                    message
                    field
                }
            }
        }
    `);

    const [disableMaintenanceMode, isDisabling] = useMutation<MaintenanceSettingsDisableMutation>(graphql`
        mutation MaintenanceSettingsDisableMutation($input: DisableMaintenanceModeInput!) {
            disableMaintenanceMode(input: $input) {
                problems {
                    message
                    field
                }
            }
        }
    `);

    const handleProblemsOrSuccess = (problems: any[], successMessage: string, newState: boolean, maintenanceModeData?: any) => {
        if (problems.length) {
            setError({
                severity: 'warning',
                message: problems.map((p: any) => p.message).join(', ')
            });
        } else {
            enqueueSnackbar(successMessage, { variant: 'success' });
            setIsMaintenanceModeEnabled(newState);
            setCurrentMaintenanceMode(maintenanceModeData || null);
        }
    };

    const onToggleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        setIsMaintenanceModeEnabled(event.target.checked);
        setError(undefined);
    };

    const onSave = () => {
        setError(undefined);

        const baseConfig = {
            variables: { input: {} },
            onError: (error: Error) => {
                setError({
                    severity: 'error' as const,
                    message: `Unexpected error occurred: ${error.message}`
                });
            }
        };

        if (isMaintenanceModeEnabled) {
            enableMaintenanceMode({
                ...baseConfig,
                onCompleted: (response: any) => {
                    handleProblemsOrSuccess(
                        response.enableMaintenanceMode.problems,
                        'Maintenance mode enabled',
                        true,
                        response.enableMaintenanceMode.maintenanceMode
                    );
                }
            });
        } else {
            disableMaintenanceMode({
                ...baseConfig,
                onCompleted: (response: any) => {
                    handleProblemsOrSuccess(
                        response.disableMaintenanceMode.problems,
                        'Maintenance mode disabled',
                        false,
                        null
                    );
                }
            });
        }
    };

    const hasChanges = isMaintenanceModeEnabled !== !!currentMaintenanceMode;
    const isLoading = isEnabling || isDisabling;

    return (
        <Box>
            <Typography variant="subtitle1" gutterBottom>
                Maintenance Mode
            </Typography>

            <Box sx={{ mt: 2 }}>
                <Typography variant="body2" color="textSecondary" paragraph>
                    When maintenance mode is enabled, the system will go into read-only mode
                    and users will not be able to make changes or perform write operations.
                </Typography>

                {error && (
                    <Alert severity={error.severity}>
                        {error.message}
                    </Alert>
                )}

                <FormControlLabel
                    control={
                        <Switch
                            checked={isMaintenanceModeEnabled}
                            onChange={onToggleChange}
                            disabled={isLoading}
                            color="primary"
                        />
                    }
                    label="Enable Maintenance Mode"
                />

                {currentMaintenanceMode && (
                    <Typography variant="body2" color="textSecondary" component="span" sx={{ mt: 1, display: 'block' }}>
                        Enabled by {currentMaintenanceMode.createdBy} on{' '}
                        <Timestamp timestamp={currentMaintenanceMode.metadata.createdAt} format="absolute" />
                    </Typography>
                )}
            </Box>

            <LoadingButton
                loading={isLoading}
                disabled={!hasChanges}
                variant="outlined"
                color="primary"
                size="small"
                onClick={onSave}
                sx={{ mt: 4 }}
            >
                Save Changes
            </LoadingButton>
        </Box>
    );
}

export default MaintenanceSettings;
