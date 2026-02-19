import { Box, Typography } from '@mui/material';
import { AdapterMoment } from '@mui/x-date-pickers/AdapterMoment';
import { DateTimePicker } from '@mui/x-date-pickers/DateTimePicker';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import moment, { Moment } from 'moment';

const MIN_EXPIRATION_DAYS = 1;

interface Props {
    value: Moment | null;
    onChange: (value: Moment | null) => void;
    maxExpirationDays: number;
    disabled?: boolean;
    label?: string;
    width?: number | string;
}

export function isExpirationInvalid(value: Moment | null | undefined, maxExpirationDays: number): boolean {
    if (!value?.isValid()) return false; // null is valid (uses default)
    const min = moment().add(MIN_EXPIRATION_DAYS, 'days');
    const max = moment().add(maxExpirationDays, 'days');
    return value.isBefore(min) || value.isAfter(max);
}

function ExpirationDateTimePicker({ value, onChange, maxExpirationDays, disabled, label = 'Secret Expiration', width = 300 }: Props) {
    return (
        <Box>
            <LocalizationProvider dateAdapter={AdapterMoment}>
                <DateTimePicker
                    sx={{ width }}
                    label={label}
                    slotProps={{
                        textField: { size: 'small' },
                        actionBar: { actions: ['clear', 'accept'] },
                    }}
                    minDateTime={moment().add(MIN_EXPIRATION_DAYS, 'days')}
                    maxDateTime={moment().add(maxExpirationDays, 'days')}
                    value={value}
                    onChange={onChange}
                    disabled={disabled}
                    closeOnSelect={false}
                />
            </LocalizationProvider>
            <Typography variant="caption" color="textSecondary" display="block" sx={{ mt: 0.5 }}>
                {value?.isValid()
                    ? `Secret will expire ${value.fromNow()}`
                    : `Min: ${MIN_EXPIRATION_DAYS} day, max and default: ${maxExpirationDays} days`}
            </Typography>
        </Box>
    );
}

export default ExpirationDateTimePicker;
