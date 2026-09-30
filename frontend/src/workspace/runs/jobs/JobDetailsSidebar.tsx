import { Box, Chip, Stack, Typography } from '@mui/material';
import Drawer from '../../../common/Drawer';
import TRNButton from '../../../common/TRNButton';
import Link from '../../../routes/Link';
import RunStatusChip from '../RunStatusChip';
import JobDetailsTimeline from './JobDetailsTimeline';
import JobStatusChip from './JobStatusChip';
import { JOB_TYPE_LABELS } from './jobUtils';
import { JobDetailsQuery } from './__generated__/JobDetailsQuery.graphql';

type Job = NonNullable<JobDetailsQuery['response']['node']>;

export const JobSidebarWidth = 320;

interface Props {
    job: Job;
    status: string;
    open: boolean;
    temporary: boolean;
    onClose: () => void;
}

function runnerValue(job: Job) {
    if (job.runner) {
        if (job.runner.type === 'shared') {
            return (
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap' }}>
                    <Typography variant="body2">{job.runner.name}</Typography>
                    <Chip size="small" label="Shared" />
                </Box>
            );
        }

        return (
            <Link color="secondary" sx={{ fontWeight: 500, wordBreak: 'break-all' }} to={`/groups/${job.runner.groupPath}/-/runners/${job.runner.id}`}>
                {job.runner.name}
            </Link>
        );
    }

    // runner is null when it was deleted or the caller lacks access to it; fall back to the plain
    // runnerPath string (always available) without claiming it was deleted.
    if (job.runnerPath) {
        return <Typography variant="body2" sx={{ wordBreak: 'break-all' }}>{job.runnerPath}</Typography>;
    }

    return <Typography variant="body2" color="textSecondary">Not assigned</Typography>;
}

function JobDetailsSidebar({ job, status, open, temporary, onClose }: Props) {
    return (
        <Drawer
            width={JobSidebarWidth}
            temporary={temporary}
            variant={temporary ? 'temporary' : 'permanent'}
            open={open}
            hideBackdrop={false}
            anchor="right"
            onClose={onClose}
        >
            <Box padding={2}>
                <Box marginBottom={3} display="flex" alignItems="center" justifyContent="space-between" gap={1}>
                    <Typography variant="h6">Job Details</Typography>
                    <TRNButton size="small" trn={job.metadata!.trn} />
                </Box>

                <Box marginBottom={3}>
                    <Typography sx={{ marginBottom: 1 }}>Status</Typography>
                    <JobStatusChip status={status} />
                </Box>

                <Box marginBottom={3}>
                    <Typography sx={{ marginBottom: 1 }}>Type</Typography>
                    <Chip size="small" label={JOB_TYPE_LABELS[job.type ?? ''] ?? job.type} />
                </Box>

                {job.tags && job.tags.length > 0 && (
                    <Box marginBottom={3}>
                        <Typography sx={{ marginBottom: 1 }}>Tags</Typography>
                        <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.75 }}>
                            {job.tags.map(tag => <Chip key={tag} size="small" color="secondary" label={tag} />)}
                        </Box>
                    </Box>
                )}

                <Box marginBottom={3}>
                    <Typography sx={{ marginBottom: 1 }}>Workspace</Typography>
                    {job.workspace ? (
                        <Link
                            color="secondary"
                            title={job.workspace.fullPath}
                            sx={{
                                fontWeight: 500,
                                display: 'block',
                                maxWidth: '100%',
                                overflow: 'hidden',
                                textOverflow: 'ellipsis',
                                whiteSpace: 'nowrap',
                            }}
                            to={`/groups/${job.workspace.fullPath}`}
                        >
                            {job.workspace.fullPath.split('/').pop()}
                        </Link>
                    ) : <Typography variant="body2" color="textSecondary">—</Typography>}
                </Box>

                {job.run && (
                    <Box marginBottom={3}>
                        <Typography sx={{ marginBottom: 1 }}>Run</Typography>
                        <Stack direction="row" spacing={1} alignItems="center" flexWrap="wrap" useFlexGap>
                            <Link color="secondary" sx={{ fontWeight: 500 }} to={`/groups/${job.run.workspace.fullPath}/-/runs/${job.run.id}`}>
                                {job.run.id.substring(0, 8)}…
                            </Link>
                            <RunStatusChip status={job.run.status} hasAdvisoryFailures={job.run.hasAdvisoryFailures} />
                        </Stack>
                    </Box>
                )}

                <Box marginBottom={3}>
                    <Typography sx={{ marginBottom: 1 }}>Runner</Typography>
                    {runnerValue(job)}
                </Box>

                <Box marginBottom={3}>
                    <Typography sx={{ marginBottom: 1 }}>Timeline</Typography>
                    <JobDetailsTimeline job={job} />
                </Box>
            </Box>
        </Drawer>
    );
}

export default JobDetailsSidebar;
