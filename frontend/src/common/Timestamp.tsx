import { Box, Tooltip } from '@mui/material';
import Typography, { TypographyProps } from '@mui/material/Typography';
import { useMemo } from 'react';
import moment from 'moment-timezone';

interface Props {
    timestamp: string;
    format?: 'relative' | 'absolute';
    tooltip?: string;
}

function Timestamp({ timestamp, format, tooltip, ...other }: Props & TypographyProps) {
    const formattedTimestamp = useMemo(() => {
        const zoneName = moment.tz(moment.tz.guess()).zoneName();
        return `${moment(timestamp as moment.MomentInput).format('MMMM Do YYYY, h:mm:ss A')} ${zoneName}`;
    }, [timestamp]);

    const relativeTimestamp = moment(timestamp as moment.MomentInput).fromNow();

    return (
        <Tooltip title={tooltip || (format === 'absolute' ? relativeTimestamp : formattedTimestamp)}>
            <Box component="span" display="inline-block">
                <Typography variant="inherit" color="inherit" {...other}>
                    {format === 'absolute' ? formattedTimestamp : relativeTimestamp}
                </Typography>
            </Box>
        </Tooltip>
    );
}

export default Timestamp;
