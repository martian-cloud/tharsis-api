import SubjectIcon from '@mui/icons-material/Subject';
import { Box, Paper, Typography } from '@mui/material';
import { useTheme } from '@mui/material/styles';
import { Suspense } from 'react';
import JobLogs from './JobLogs';

function JobDetailsLogsCard({ jobId }: { jobId: string }) {
    const theme = useTheme();

    return (
        <Paper variant="outlined" sx={{ borderRadius: 2, overflow: 'hidden' }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, px: 3, py: 2 }}>
                <SubjectIcon sx={{ color: theme.palette.text.secondary, fontSize: 18 }} />
                <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>Logs</Typography>
            </Box>
            <Suspense fallback={null}>
                <JobLogs jobId={jobId} />
            </Suspense>
        </Paper>
    );
}

export default JobDetailsLogsCard;
