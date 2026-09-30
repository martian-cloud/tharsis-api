import { Box, Button, CircularProgress, Typography } from '@mui/material';
import { ResponsiveTable } from '../../common/ResponsiveTable';
import RunJobListItem from './RunJobListItem';
import type { RunJobDialog_jobs$data } from './__generated__/RunJobDialog_jobs.graphql';

type Job = RunJobDialog_jobs$data[number];

const COLUMNS = [
    { label: 'Status' },
    { label: 'ID' },
    { label: 'Runner' },
    { label: 'Tags' },
    { label: 'Duration' },
    { label: 'Created' },
];

interface Props {
    jobs: readonly Job[];
    selectedJobId: string | null;
    hasMore: boolean;
    loadingMore: boolean;
    onSelectJob: (jobId: string) => void;
    onLoadMore: () => void;
}

function RunJobList({ jobs, hasMore, loadingMore, onSelectJob, onLoadMore }: Props) {
    return (
        <Box>
            {jobs.length === 0 ? (
                <Box sx={{ p: 2 }}>
                    <Typography variant="body2" color="textSecondary">No jobs</Typography>
                </Box>
            ) : (
                <ResponsiveTable columns={COLUMNS} ariaLabel="run jobs">
                    {jobs.map(job => (
                        <RunJobListItem
                            key={job.id}
                            job={job}
                            onClick={() => onSelectJob(job.id)}
                        />
                    ))}
                </ResponsiveTable>
            )}
            {hasMore && (
                <Box sx={{ p: 1, display: 'flex', justifyContent: 'center' }}>
                    <Button
                        size="small"
                        color="inherit"
                        disabled={loadingMore}
                        onClick={onLoadMore}
                        startIcon={loadingMore ? <CircularProgress size={14} color="inherit" /> : undefined}
                    >
                        {loadingMore ? 'Loading...' : 'Load more'}
                    </Button>
                </Box>
            )}
        </Box>
    );
}

export default RunJobList;
