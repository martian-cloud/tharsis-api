import { Chip, Stack, Typography } from '@mui/material';
import humanizeDuration from 'humanize-duration';
import { ResponsiveRow } from '../../common/ResponsiveTable';
import Timestamp from '../../common/Timestamp';
import JobStatusChip from './jobs/JobStatusChip';
import { jobDuration } from './jobs/jobUtils';
import type { RunJobDialog_jobs$data } from './__generated__/RunJobDialog_jobs.graphql';

type Job = RunJobDialog_jobs$data[number];

interface Props {
    job: Job;
    onClick: () => void;
}

function RunJobListItem({ job, onClick }: Props) {
    const duration = jobDuration(job.timestamps);

    return (
        <ResponsiveRow
            onClick={onClick}
            cells={[
                {
                    primary: true,
                    content: <JobStatusChip status={job.status} />,
                },
                {
                    label: 'ID',
                    content: (
                        <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                            {job.id.substring(0, 8)}...
                        </Typography>
                    ),
                },
                {
                    label: 'Runner',
                    content: (
                        <>
                            <Typography variant="body2">
                                {job.runner?.name ?? job.runnerPath ?? '—'}
                            </Typography>
                        </>
                    ),
                },
                {
                    label: 'Tags',
                    content: job.tags && job.tags.length > 0 ? (
                        <Stack direction="row" flexWrap="wrap" gap={0.5}>
                            {job.tags.map((tag: string) => (
                                <Chip key={tag} size="small" color="secondary" label={tag} />
                            ))}
                        </Stack>
                    ) : (
                        <Typography variant="caption" color="textSecondary">—</Typography>
                    ),
                },
                {
                    label: 'Duration',
                    content: duration ? (
                        <Typography variant="body2">
                            {humanizeDuration(duration.asMilliseconds(), { maxDecimalPoints: 1 })}
                        </Typography>
                    ) : (
                        <Typography variant="caption" color="textSecondary">—</Typography>
                    ),
                },
                {
                    label: 'Created',
                    content: <Timestamp variant="body2" timestamp={job.metadata.createdAt as string} />,
                },
            ]}
        />
    );
}

export default RunJobListItem;
