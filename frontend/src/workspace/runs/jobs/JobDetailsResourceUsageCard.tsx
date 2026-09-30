import SpeedIcon from '@mui/icons-material/Speed';
import { Box, Typography } from '@mui/material';
import { useTheme } from '@mui/material/styles';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { Card, CardTitle } from './JobDetailsCard';
import JobResourceUsageMetrics from './JobResourceUsageMetrics';
import { JobDetailsResourceUsageCardFragment_job$key } from './__generated__/JobDetailsResourceUsageCardFragment_job.graphql';

// JobDetailsResourceUsageCard frames a job's resource usage metrics in a titled card for the job details page.
function JobDetailsResourceUsageCard({ fragmentRef }: { fragmentRef: JobDetailsResourceUsageCardFragment_job$key }) {
    const theme = useTheme();

    const job = useFragment(
        graphql`
        fragment JobDetailsResourceUsageCardFragment_job on Job {
            resourceUsageMetrics {
                totalCpuTimeMs
            }
            ...JobResourceUsageMetricsFragment_job
        }
        `,
        fragmentRef
    );

    if (!job.resourceUsageMetrics) {
        return null;
    }

    return (
        <Card sx={{ mb: 3, height: 'auto' }}>
            <CardTitle
                action={
                    <Typography variant="caption" sx={{ color: theme.palette.text.secondary }}>
                        Collected from the job's runtime container
                    </Typography>
                }
            >
                <Box component="span" sx={{ display: 'inline-flex', alignItems: 'center', gap: 1.5, verticalAlign: 'middle' }}>
                    <SpeedIcon sx={{ color: theme.palette.text.secondary, fontSize: 18 }} />
                    Resource usage
                </Box>
            </CardTitle>
            <JobResourceUsageMetrics fragmentRef={job} />
        </Card>
    );
}

export default JobDetailsResourceUsageCard;
