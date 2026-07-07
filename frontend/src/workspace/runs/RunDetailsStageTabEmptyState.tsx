import { Paper, Typography } from '@mui/material';

interface Props {
    message: string;
}

// RunDetailsStageTabEmptyState is the placeholder shown inside an always-visible tab
// whose content is not available for the run's current state (e.g. logs before the
// stage has started, or plan changes before the plan finishes).
function RunDetailsStageTabEmptyState({ message }: Props) {
    return (
        <Paper
            variant="outlined"
            sx={{
                minHeight: 100,
                display: 'flex',
                flexDirection: 'column',
                justifyContent: 'center',
            }}
        >
            <Typography color="textSecondary" align="center">
                {message}
            </Typography>
        </Paper>
    );
}

export default RunDetailsStageTabEmptyState;
