import { Box, Checkbox, Divider, Paper, Typography, FormControlLabel } from '@mui/material';

export interface CustomEvents {
    failedRun: boolean;
    serviceAccountSecretExpiration: boolean;
}

interface EventConfig {
    key: keyof CustomEvents;
    label: string;
    description: string;
}

// Single source of truth: add new events here
const EVENT_CONFIG: Record<string, EventConfig[]> = {
    'Run Events': [
        { key: 'failedRun', label: 'Failed Run', description: 'Receive a notification when a run fails' },
    ],
    'Service Account Events': [
        { key: 'serviceAccountSecretExpiration', label: 'Secret Expiration', description: 'Receive a notification when a service account client secret is about to expire' },
    ],
};

// Default values derived from EVENT_CONFIG
const DEFAULT_CUSTOM_EVENTS = Object.fromEntries(
    Object.values(EVENT_CONFIG).flat().map(e => [e.key, false])
) as unknown as CustomEvents;

interface Props {
    events: Partial<CustomEvents> | null | undefined;
    onChange: (events: CustomEvents) => void;
    disabled?: boolean;
}

function CustomNotificationPreference({ events, onChange, disabled = false }: Props) {
    const resolvedEvents: CustomEvents = { ...DEFAULT_CUSTOM_EVENTS, ...events };

    return (
        <Paper variant="outlined" sx={{ p: 2 }}>
            <Typography variant="subtitle2" gutterBottom>
                Custom Notification Preference
            </Typography>
            <Divider sx={{ mb: 2 }} />
            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                Select specific events you want to be notified about.
            </Typography>

            {Object.entries(EVENT_CONFIG).map(([category, categoryEvents]) => (
                <Box key={category} sx={{ mb: 2 }}>
                    <Typography variant="subtitle2" sx={{ mb: 2 }}>
                        {category}
                    </Typography>
                    {categoryEvents.map(event => (
                        <FormControlLabel
                            key={event.key}
                            control={
                                <Checkbox
                                    checked={resolvedEvents[event.key]}
                                    onChange={(e) => onChange({ ...resolvedEvents, [event.key]: e.target.checked })}
                                    color="primary"
                                    size="small"
                                    disabled={disabled}
                                />
                            }
                            label={
                                <Box>
                                    <Typography variant="body2">{event.label}</Typography>
                                    <Typography variant="caption" color="text.secondary">
                                        {event.description}
                                    </Typography>
                                </Box>
                            }
                        />
                    ))}
                </Box>
            ))}
        </Paper>
    );
}

export default CustomNotificationPreference;
