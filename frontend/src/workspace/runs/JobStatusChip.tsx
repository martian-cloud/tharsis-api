import { Chip } from '@mui/material';
import blue from '@mui/material/colors/blue';
import green from '@mui/material/colors/green';
import grey from '@mui/material/colors/grey';
import orange from '@mui/material/colors/orange';
import { Link as RouterLink } from 'react-router-dom';

export const JOB_STATUS_TYPES = {
    "finished": {
        label: "Completed",
        color: green[500]
    },
    "running": {
        label: "In Progress",
        color: blue[500]
    },
    "pending": {
        label: "Pending",
        color: orange[500]
    },
    "queued": {
        label: "Queued",
        color: orange[500]
    },
} as any;

interface Props {
    to?: string
    status: string
    onClick?: () => void
}

function JobStatusChip({ to, status, onClick }: Props) {
    const type = JOB_STATUS_TYPES[status] ?? { label: 'Unknown', color: grey[500] }
    return to ? (
        <Chip
            to={to}
            component={RouterLink}
            clickable
            size="small"
            variant="outlined"
            label={type.label}
            sx={{ color: type.color, borderColor: type.color, fontWeight: 500 }}
        />
    ) : (
        <Chip
            onClick={onClick}
            clickable
            size="small"
            variant="outlined"
            label={type.label}
            sx={{ color: type.color, borderColor: type.color, fontWeight: 500 }}
        />
    );
}

export default JobStatusChip
