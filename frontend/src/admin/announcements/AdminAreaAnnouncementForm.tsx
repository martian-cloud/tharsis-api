import { Alert, Box, Checkbox, Divider, FormControl, FormControlLabel, InputLabel, Link, MenuItem, Select, TextField, Typography } from '@mui/material';
import { AdapterMoment } from '@mui/x-date-pickers/AdapterMoment';
import { DateTimePicker } from '@mui/x-date-pickers/DateTimePicker';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import moment, { Moment } from 'moment';
import { useMemo } from 'react';
import { MutationError } from "../../common/error";
import Timestamp from '../../common/Timestamp';
import AnnouncementAlert from '../../common/AnnouncementAlert';
import { AnnouncementType } from '../../common/__generated__/AnnouncementBannerQuery.graphql';

export interface FormData {
    message: string;
    type: AnnouncementType;
    dismissible: boolean;
    startTime: Moment | null;
    endTime: Moment | null;
}

interface Props {
    data: FormData;
    onChange: (data: FormData) => void;
    editMode?: boolean;
    error?: MutationError;
}

const TYPE_OPTIONS: { value: AnnouncementType; label: string; description: string }[] = [
    { value: 'INFO', label: 'Info', description: 'General information or updates' },
    { value: 'SUCCESS', label: 'Success', description: 'Positive news or completed actions' },
    { value: 'WARNING', label: 'Warning', description: 'Important notices requiring attention' },
    { value: 'ERROR', label: 'Error', description: 'Critical issues or system problems' }
];

function AdminAreaAnnouncementForm({ data, onChange, error }: Props) {
    const onStartTimeChange = (date: Moment | null) => {
        onChange({ ...data, startTime: date });
    };

    const onEndTimeChange = (date: Moment | null) => {
        onChange({ ...data, endTime: date });
    };

    const onDismissibleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        onChange({ ...data, dismissible: event.target.checked });
    };

    const showPreview = useMemo(() => {
        return data.message.trim().length > 0;
    }, [data.message]);

    return (
        <Box>
            {error && <Alert sx={{ mb: 2 }} severity={error.severity}>
                {error.message}
            </Alert>}

            {showPreview && (
                <Box sx={{ mb: 3 }}>
                    <Typography variant="subtitle1" gutterBottom>Preview</Typography>
                    <AnnouncementAlert
                        id="preview"
                        message={data.message}
                        type={data.type}
                        dismissible={data.dismissible}
                        onDismiss={() => { /* Preview only - no action */ }}
                    />
                </Box>
            )}

            <Typography mt={2} variant="subtitle1" gutterBottom>Details</Typography>
            <Divider />
            <Box sx={{ mt: 2, mb: 2 }}>
                <TextField
                    fullWidth
                    multiline
                    minRows={3}
                    autoComplete="off"
                    size="small"
                    label="Message"
                    placeholder="Enter your announcement message..."
                    value={data.message}
                    onChange={event => onChange({ ...data, message: event.target.value })}
                    sx={{ mb: 1 }}
                />
                <Typography
                    variant="caption"
                    color="textSecondary"
                    sx={{ mb: 2, display: 'block' }}
                >
                    Select markdown features are supported. Learn more about{' '}
                    <Link
                        href='https://commonmark.org/help/'
                        target='_blank'
                        rel='noopener noreferrer'
                        underline='hover'
                    >
                        markdown
                    </Link>.
                </Typography>

                <FormControl fullWidth margin="normal" size="small" sx={{ mb: 2 }}>
                    <InputLabel>Type</InputLabel>
                    <Select
                        value={data.type}
                        label="Type"
                        onChange={event => onChange({ ...data, type: event.target.value as AnnouncementType })}
                    >
                        {TYPE_OPTIONS.map(option => (
                            <MenuItem key={option.value} value={option.value}>
                                <Box>
                                    <Typography variant="body2">{option.label}</Typography>
                                    <Typography variant="caption" color="textSecondary">
                                        {option.description}
                                    </Typography>
                                </Box>
                            </MenuItem>
                        ))}
                    </Select>
                </FormControl>

                <Box sx={{ mt: 3, mb: 2 }}>
                    <Typography variant="body2" gutterBottom>
                        Start Time (Optional)
                    </Typography>
                    <LocalizationProvider dateAdapter={AdapterMoment}>
                        <DateTimePicker
                            sx={{ width: '100%' }}
                            slotProps={{
                                textField: { size: 'small' },
                                actionBar: { actions: ['clear', 'accept'] },
                            }}
                            disablePast={true}
                            value={data.startTime}
                            onChange={onStartTimeChange}
                            closeOnSelect={false}
                        />
                    </LocalizationProvider>
                    {data.startTime?.isValid() && (
                        <Typography mt={1} variant="caption" color="textSecondary">
                            This announcement {data.startTime.isBefore(moment()) ? 'became active' : 'will become active'}{' '}
                            <Timestamp
                                timestamp={data.startTime.toISOString()}
                                format="relative"
                            />
                        </Typography>
                    )}
                    {!data.startTime && (
                        <Typography mt={1} variant="caption" color="textSecondary">
                            This announcement will become active immediately when saved.
                        </Typography>
                    )}
                </Box>

                <Box sx={{ mt: 2, mb: 2 }}>
                    <Typography variant="body2" gutterBottom>
                        End Time (Optional)
                    </Typography>
                    <LocalizationProvider dateAdapter={AdapterMoment}>
                        <DateTimePicker
                            sx={{ width: '100%' }}
                            slotProps={{
                                textField: { size: 'small' },
                                actionBar: { actions: ['clear', 'accept'] },
                            }}
                            minDateTime={data.startTime || moment()}
                            value={data.endTime}
                            onChange={onEndTimeChange}
                            closeOnSelect={false}
                        />
                    </LocalizationProvider>
                    {data.endTime?.isValid() && (
                        <Typography mt={1} variant="caption" color="textSecondary">
                            This announcement {data.endTime.isBefore(moment()) ? 'expired' : 'will expire'}{' '}
                            <Timestamp
                                timestamp={data.endTime.toISOString()}
                                format="relative"
                            />
                        </Typography>
                    )}
                    {!data.endTime && (
                        <Typography mt={1} variant="caption" color="textSecondary">
                            This announcement will remain active indefinitely until manually deleted.
                        </Typography>
                    )}
                </Box>

                <Box sx={{ mt: 3 }}>
                    <FormControlLabel
                        control={
                            <Checkbox
                                checked={data.dismissible}
                                onChange={onDismissibleChange}
                                color="primary"
                            />
                        }
                        label={
                            <Box>
                                <Typography variant="body2">
                                    Allow users to dismiss this announcement
                                </Typography>
                                <Typography variant="caption" color="textSecondary">
                                    {data.dismissible
                                        ? "Users can close this announcement and won't see it again"
                                        : "This announcement will always be visible to users during its active period"
                                    }
                                </Typography>
                            </Box>
                        }
                    />
                </Box>
            </Box>
        </Box>
    );
}

export default AdminAreaAnnouncementForm;
