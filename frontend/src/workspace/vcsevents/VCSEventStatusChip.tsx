import { Chip } from '@mui/material';
import { useTheme } from '@mui/material/styles';

interface Props {
    // The VCS event status as it comes off the wire: pending, finished or errored.
    status: string
}

function VCSEventStatusChip({ status }: Props) {
    const theme = useTheme();

    const colors: Record<string, string> = {
        finished: theme.palette.success.main,
        errored: theme.palette.error.main,
        pending: theme.palette.text.secondary,
    };
    const color = colors[status] ?? theme.palette.text.secondary;

    return (
        <Chip
            size="small"
            variant="outlined"
            label={status}
            sx={{ color, borderColor: color, fontWeight: 500 }}
        />
    );
}

export default VCSEventStatusChip;
