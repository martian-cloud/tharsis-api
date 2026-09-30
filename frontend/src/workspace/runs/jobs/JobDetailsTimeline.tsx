import { Box, Typography } from '@mui/material';
import { useTheme } from '@mui/material/styles';
import { shortDuration } from '../../../common/duration';
import Timestamp from '../../../common/Timestamp';
import { humanizeDurationBetween, jobDuration, useJobStatusColor } from './jobUtils';
import { JobDetailsQuery } from './__generated__/JobDetailsQuery.graphql';

type Job = NonNullable<JobDetailsQuery['response']['node']>;
type TimelineState = 'done' | 'current' | 'pending';

// TimelineStep is one phase on the vertical timeline: a dot, a connector to the next, label + time.
function TimelineStep({ label, timestamp, state, statusColor, last }: {
    label: string;
    timestamp: string | null | undefined;
    state: TimelineState;
    statusColor: string;
    last?: boolean;
}) {
    const theme = useTheme();

    const dotSx =
        state === 'done'
            ? { backgroundColor: statusColor, border: 'none' }
            : state === 'current'
                ? {
                    backgroundColor: theme.palette.secondary.main,
                    border: 'none',
                    animation: 'jobTimelinePulse 1.5s ease-in-out infinite',
                    '@keyframes jobTimelinePulse': {
                        '0%, 100%': { opacity: 1 },
                        '50%': { opacity: 0.25 },
                    },
                }
                : { backgroundColor: 'transparent', border: `2px solid ${theme.palette.divider}` };

    const active = state !== 'pending';

    return (
        <Box sx={{ display: 'flex', gap: 1.5 }}>
            <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
                <Box sx={{ width: 10, height: 10, borderRadius: '50%', flexShrink: 0, mt: '4px', boxSizing: 'border-box', ...dotSx }} />
                {!last && <Box sx={{ width: 2, flex: 1, minHeight: 32, backgroundColor: theme.palette.divider, my: '2px' }} />}
            </Box>
            <Box sx={{ pb: last ? 0 : '16px', minWidth: 0 }}>
                <Typography variant="body2" sx={{ fontWeight: 500, color: active ? theme.palette.text.primary : theme.palette.text.secondary }}>
                    {label}
                </Typography>
                <Typography variant="caption" sx={{ color: theme.palette.text.secondary }}>
                    {timestamp ? <Timestamp timestamp={timestamp} format="absolute" /> : '—'}
                </Typography>
            </Box>
        </Box>
    );
}

function DurationStat({ label, value }: { label: string; value: string }) {
    const theme = useTheme();

    return (
        <Box>
            <Typography variant="caption" sx={{ display: 'block', color: theme.palette.text.secondary, mb: '2px' }}>
                {label}
            </Typography>
            <Typography variant="body2" sx={{ fontWeight: 600 }}>{value}</Typography>
        </Box>
    );
}

function JobDetailsTimeline({ job }: { job: Job }) {
    const theme = useTheme();

    const ts = job.timestamps!;
    const createdAt = job.metadata!.createdAt;
    const runDuration = jobDuration(ts);
    const queuedDuration = humanizeDurationBetween(ts.pendingAt, ts.runningAt);
    const totalDuration = humanizeDurationBetween(createdAt, ts.finishedAt);

    // Reached flags per phase, in order. Created is reached once the job exists.
    const reached = [true, !!ts.pendingAt, !!ts.runningAt, !!ts.finishedAt];
    const status = (job.status ?? '').toLowerCase();
    const terminal = ['finished', 'failed', 'canceled'].includes(status);
    const currentIdx = terminal ? -1 : reached.lastIndexOf(true);
    const stateFor = (i: number): TimelineState =>
        i === currentIdx ? 'current' : reached[i] ? 'done' : 'pending';

    // Completed dots use the same color as the status pill so the timeline matches the job's status.
    const statusColor = useJobStatusColor(status);

    return (
        <Box>
            <Box>
                <TimelineStep label="Created" timestamp={createdAt as string} state={stateFor(0)} statusColor={statusColor} />
                <TimelineStep label="Pending" timestamp={ts.pendingAt as string} state={stateFor(1)} statusColor={statusColor} />
                <TimelineStep label="Running" timestamp={ts.runningAt as string} state={stateFor(2)} statusColor={statusColor} />
                <TimelineStep label="Finished" timestamp={ts.finishedAt as string} state={stateFor(3)} statusColor={statusColor} last />
            </Box>
            <Box sx={{
                display: 'flex',
                justifyContent: 'space-between',
                gap: 2,
                mt: '16px',
                pt: '16px',
                borderTop: `1px solid ${theme.palette.divider}`,
            }}>
                <DurationStat label="Queued" value={queuedDuration ?? '—'} />
                <DurationStat label="Running" value={runDuration ? shortDuration(runDuration.asMilliseconds()) : '—'} />
                <DurationStat label="Total" value={totalDuration ?? '—'} />
            </Box>
        </Box>
    );
}

export default JobDetailsTimeline;
