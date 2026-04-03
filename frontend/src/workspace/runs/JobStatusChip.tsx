import { Chip } from '@mui/material';
import { useTheme } from '@mui/material/styles';
import { Link as RouterLink } from 'react-router-dom';

const JOB_STATUS_MAP: Record<string, { label: string }> = {
    finished: { label: 'Completed' },
    running: { label: 'In Progress' },
    pending: { label: 'Pending' },
    queued: { label: 'Queued' },
};

interface Props {
    to?: string
    status: string
    onClick?: () => void
}

function JobStatusChip({ to, status, onClick }: Props) {
    const theme = useTheme();
    const entry = JOB_STATUS_MAP[status];
    const color = entry
        ? theme.palette.runStatus[status as keyof typeof theme.palette.runStatus]
        : theme.palette.runStatus.unknown;
    const label = entry?.label ?? 'Unknown';

    return to ? (
        <Chip
            to={to}
            component={RouterLink}
            clickable
            size="small"
            variant="outlined"
            label={label}
            sx={{ color, borderColor: color, fontWeight: 500 }}
        />
    ) : (
        <Chip
            onClick={onClick}
            clickable
            size="small"
            variant="outlined"
            label={label}
            sx={{ color, borderColor: color, fontWeight: 500 }}
        />
    );
}

export default JobStatusChip
