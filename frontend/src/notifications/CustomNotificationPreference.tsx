import { Box, Checkbox, Divider, Paper, Typography, FormControlLabel } from '@mui/material';

interface Props {
    failedRun: boolean;
    onChange: (settings: { failedRun: boolean }) => void;
    disabled?: boolean;
}

function CustomNotificationPreference({ failedRun, onChange, disabled = false }: Props) {
    return (
        <Paper variant="outlined" sx={{ p: 2 }}>
            <Typography variant="subtitle2" gutterBottom>
                Custom Notification Preference
            </Typography>
            <Divider sx={{ mb: 2 }} />
            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                Select specific events you want to be notified about.
            </Typography>

            <Box sx={{ mb: 2 }}>
                <Typography variant="subtitle2" sx={{ mb: 2 }}>
                    Run Events
                </Typography>
                <FormControlLabel
                    control={
                        <Checkbox
                            checked={failedRun}
                            onChange={(event) => onChange({ failedRun: event.target.checked })}
                            color="primary"
                            size="small"
                            disabled={disabled}
                        />
                    }
                    label={
                        <Box>
                            <Typography variant="body2">Failed Run</Typography>
                            <Typography variant="caption" color="text.secondary">
                                Receive a notification when a run fails
                            </Typography>
                        </Box>
                    }
                />
            </Box>
        </Paper>
    );
}

export default CustomNotificationPreference
