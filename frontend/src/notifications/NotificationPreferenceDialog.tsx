import { useState, useMemo } from 'react';
import { Alert, Dialog, DialogTitle, DialogContent, DialogActions, Button, Box, Typography, Checkbox, FormControlLabel, FormControl, InputLabel, Select, MenuItem, SelectChangeEvent } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useMutation } from 'react-relay/hooks';
import { notificationOptions } from './NotificationButton';
import { MutationError } from '../common/error';
import { LoadingButton } from '@mui/lab';
import { InheritedMessage, Preference } from './NotificationButton';
import CustomNotificationPreference from './CustomNotificationPreference';
import { NotificationPreferenceDialogMutation, UserNotificationPreferenceScope, UserNotificationPreferenceCustomEventsInput } from './__generated__/NotificationPreferenceDialogMutation.graphql';

const NOTIFICATION_SCOPE_CUSTOM = 'CUSTOM' as UserNotificationPreferenceScope;

interface Props {
    onClose: () => void;
    path: string | null;
    preferenceData: Preference
    isGlobalPreference?: boolean;
    onPreferenceUpdated?: (pref: Preference) => void;
}

interface FormData {
    scope: UserNotificationPreferenceScope;
    inherit: boolean;
    customEvents: UserNotificationPreferenceCustomEventsInput | null | undefined;
}

function NotificationPreferenceDialog({ onClose, path, preferenceData, isGlobalPreference, onPreferenceUpdated }: Props) {
    const [error, setError] = useState<MutationError>();
    const [formData, setFormData] = useState<FormData>({
        scope: preferenceData.scope,
        inherit: preferenceData.inherited,
        customEvents: preferenceData.customEvents
    });

    const [commit, isInFlight] = useMutation<NotificationPreferenceDialogMutation>(
        graphql`
        mutation NotificationPreferenceDialogMutation($input: SetUserNotificationPreferenceInput!) {
            setUserNotificationPreference(input: $input) {
                preference {
                    scope
                    inherited
                    global
                    namespacePath
                    customEvents {
                        failedRun
                    }
                }
                problems {
                    message
                    field
                    type
                }
            }
        }`
    );

    const onSave = () => {
        commit({
            variables: {
                input: {
                    namespacePath: isGlobalPreference ? null : path,
                    inherit: isGlobalPreference ? null : (formData.inherit ? true : null),
                    scope: isGlobalPreference ? formData.scope : (formData.inherit ? null : formData.scope),
                    customEvents: formData.scope === NOTIFICATION_SCOPE_CUSTOM ? formData.customEvents : null
                }
            },
            onCompleted: data => {
                if (data.setUserNotificationPreference.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.setUserNotificationPreference.problems.map((problem: { message: any; }) => problem.message).join('; ')
                    });
                } else if (!data.setUserNotificationPreference.preference) {
                    setError({
                        severity: 'error',
                        message: "Unexpected error occurred"
                    });
                }
                else {
                    // Call the callback with the updated preference if provided
                    if (onPreferenceUpdated) {
                        onPreferenceUpdated(data.setUserNotificationPreference.preference);
                    }
                    onClose();
                }
            },
            onError: error => {
                setError({
                    severity: 'error',
                    message: `Unexpected error occurred: ${error.message}`
                });
            }
        })
    };

    const handleInheritedChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        setFormData({ ... formData, inherit: event.target.checked});
    };

    const handleScopeChange = (event: SelectChangeEvent) => {
        setFormData({ ... formData, scope: event.target.value as UserNotificationPreferenceScope});
    };

    const handleFailedRunChange = (settings: { failedRun: boolean }) => {
        setFormData({ ... formData, customEvents: { ... formData.customEvents, failedRun: settings.failedRun }});
    }

    const optionDescription = useMemo(() => {
        const scopeToUse = formData.inherit ? preferenceData.scope : formData.scope;
        const option = notificationOptions.find(opt => opt.label === scopeToUse);
        return option ? option.description : '';
    }, [formData.scope, formData.inherit, preferenceData.scope]);

    return (
        <Dialog
            open
            onClose={onClose}
            maxWidth="sm"
            fullWidth
        >
            <DialogTitle>Notification Preference</DialogTitle>
            {error && <Alert sx={{ m: 2 }} severity={error.severity}>
                {error.message}
            </Alert>}
            <DialogContent>
                <Box>
                    {!isGlobalPreference && (
                        <FormControlLabel
                            control={
                                <Checkbox
                                    checked={formData.inherit}
                                    onChange={handleInheritedChange}
                                    color="primary"
                                />
                            }
                            label="Inherit notification preference from parent"
                            sx={{ mb: 2 }}
                        />
                    )}

                    {!formData.inherit && (
                        <FormControl fullWidth margin="normal" size="small">
                            <InputLabel id="notification-scope-label">Notification Scope</InputLabel>
                            <Select
                                labelId="notification-scope-label"
                                id="notification-scope"
                                value={formData.scope}
                                label="Notification Scope"
                                onChange={handleScopeChange}
                            >
                                {notificationOptions.map((option) => (
                                    <MenuItem key={option.label} value={option.label}>
                                        <Box sx={{ display: 'flex', flexDirection: 'column' }}>
                                            <Typography>{option.label}</Typography>
                                        </Box>
                                    </MenuItem>
                                ))}
                            </Select>
                            <Typography variant="caption" color="text.secondary" sx={{ mt: 1 }}>
                                {optionDescription}
                            </Typography>
                        </FormControl>
                    )}

                    {formData.inherit && preferenceData.inherited && (
                        <>
                            <FormControl fullWidth margin="normal" size="small">
                                <InputLabel id="notification-scope-label">Notification Scope</InputLabel>
                                <Select
                                    labelId="notification-scope-label"
                                    id="notification-scope"
                                    value={preferenceData?.scope || 'ALL'}
                                    label="Notification Scope"
                                    disabled
                                >
                                    {notificationOptions.map((option) => (
                                        <MenuItem key={option.label} value={option.label}>
                                            <Typography>{option.label}</Typography>
                                        </MenuItem>
                                    ))}
                                </Select>
                                <Typography
                                variant="caption"
                                color="text.secondary" sx={{ mt: 1 }}>
                                    {optionDescription}
                                </Typography>
                               <InheritedMessage preferenceData={preferenceData} />
                            </FormControl>

                            {preferenceData.scope === NOTIFICATION_SCOPE_CUSTOM && (
                                <CustomNotificationPreference
                                    failedRun={preferenceData.customEvents?.failedRun || false}
                                    onChange={() => ({})} // No-op function since it's disabled
                                    disabled={true}
                                />
                            )}
                        </>
                    )}

                    {formData.inherit && !preferenceData.inherited && (
                        <Typography variant="body2" sx={{ mt: 2, fontStyle: 'italic' }}>
                            Preference will be inherited from parent when saved
                        </Typography>
                    )}

                    {!formData.inherit && formData.scope === NOTIFICATION_SCOPE_CUSTOM && (
                        <CustomNotificationPreference
                            failedRun={formData.customEvents?.failedRun || false}
                            onChange={handleFailedRunChange}
                        />
                    )}
                </Box>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose} color="inherit">Cancel</Button>
                <LoadingButton
                    loading={isInFlight}
                    variant="outlined" color="primary"
                    onClick={onSave}
                >
                    Save
                </LoadingButton>
            </DialogActions>
        </Dialog>
    );
}

export default NotificationPreferenceDialog;