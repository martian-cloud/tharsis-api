import AutoScrollIcon from '@mui/icons-material/ArrowCircleDown';
import { Alert, Box, darken, LinearProgress, Paper, ToggleButton, Tooltip, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useMemo, useRef, useState } from 'react';
import { useFragment, useLazyLoadQuery, useSubscription } from 'react-relay/hooks';
import { GraphQLSubscriptionConfig, RecordSourceProxy } from 'relay-runtime';
import Timestamp from '../../common/Timestamp';
import LogViewer from './LogViewer';
import { JobLogsFragment_logs$key } from './__generated__/JobLogsFragment_logs.graphql';
import { JobLogsQuery } from './__generated__/JobLogsQuery.graphql';
import { JobLogsSubscription, JobLogsSubscription$data } from './__generated__/JobLogsSubscription.graphql';

const query = graphql`
    query JobLogsQuery($id: String!, $startOffset: Int!, $limit: Int!) {
        node(id: $id) {
            ...on Job {
                ...JobLogsFragment_logs
            }
        }
    }
`;

const subscription = graphql`subscription JobLogsSubscription($input: JobLogStreamSubscriptionInput!) {
    jobLogStreamEvents(input: $input) {
      size
      completed
      data {
        offset
        logs
      }
    }
  }`;

interface Props {
    jobId: string
    // By default the logs scroll with the window and grow the page. Pass scrollMode="container"
    // with a height to render them as a bounded, self-scrolling box instead — used when the logs
    // are embedded inside a card rather than filling a tab.
    scrollMode?: 'window' | 'container'
    height?: number | string
}

const bytes = (str: string) => {
    const size = new Blob([str]).size;
    return size;
}

const FINAL_JOB_STATES = ['finished', 'failed', 'canceled'];

const LOG_LIMIT = 51200; // 50KB

function JobLogs(props: Props) {
    // store-or-network: render already-fetched logs from the store immediately (no extra
    // suspense flash when switching tabs or revisiting a job) and only hit the network when
    // the logs aren't cached yet. The jobLogStreamEvents subscription keeps the tail live.
    const queryData = useLazyLoadQuery<JobLogsQuery>(query, { id: props.jobId, startOffset: 0, limit: LOG_LIMIT }, { fetchPolicy: 'store-or-network' });

    return queryData.node
        // Keyed on the job so a change of job remounts rather than reusing this instance. Every
        // piece of streaming state here — the log buffer, the sizes, the completed flag, the
        // auto-scroll default and the loaded-size high-water mark that dedupes subscription
        // events — is derived from the job on mount, and a retry swaps the job in place. Resetting
        // them from an effect instead is what let the high-water mark survive a retry, which
        // silently discarded every event for the new job until it outgrew the previous one.
        ? <JobLogsContent
            key={props.jobId}
            fragmentRef={queryData.node}
            scrollMode={props.scrollMode}
            height={props.height}
        />
        : <Alert severity="error">Job not found</Alert>;
}

function JobLogsContent(props: { fragmentRef: JobLogsFragment_logs$key, scrollMode?: 'window' | 'container', height?: number | string }) {
    const theme = useTheme();
    const data = useFragment<JobLogsFragment_logs$key>(
        graphql`
        fragment JobLogsFragment_logs on Job
        {
            id
            status
            completed
            logLastUpdatedAt
            logSize
            logs(startOffset: $startOffset, limit: $limit)
        }
      `, props.fragmentRef)

    const [logs, setLogs] = useState(data.logs);
    const [currentLogSize, setCurrentLogSize] = useState(bytes(data.logs));
    const [actualLogSize, setActualLogSize] = useState(data.logSize);
    const [completed, setCompleted] = useState(data.completed);
    const [autoScroll, setAutoScroll] = useState(!FINAL_JOB_STATES.includes(data.status));
    // Tracks the byte size already appended so we can dedupe events without re-measuring
    // the whole accumulated buffer on every event. Only valid for this job, which is why the
    // component is keyed on the job id by its caller.
    const loadedSizeRef = useRef(bytes(data.logs));

    const config = useMemo<GraphQLSubscriptionConfig<JobLogsSubscription>>(() => ({
        variables: { input: { jobId: data.id, lastSeenLogSize: bytes(data.logs) } },
        subscription,
        onCompleted: () => console.log("Subscription completed"),
        onError: () => console.warn("Subscription error"),
        updater: (store: RecordSourceProxy, payload: JobLogsSubscription$data | null | undefined) => {
            if (payload) {
                const event = payload.jobLogStreamEvents;
                setCompleted(event.completed);
                // Ignore stale/replayed events so the size (and progress bar) never moves backward.
                if (event.data && event.data.logs && event.size > loadedSizeRef.current) {
                    const newLogs = event.data.logs;
                    loadedSizeRef.current = event.size;
                    setCurrentLogSize(event.size);
                    setActualLogSize(event.size);
                    setLogs(prevLogs => prevLogs + newLogs);
                }
            }
        }
    }), [data.id]);
    useSubscription<JobLogsSubscription>(config);

    const loadedPercent = useMemo(() => (currentLogSize / actualLogSize) * 100, [currentLogSize, actualLogSize]);

    return (
        <Box>
            <Paper square>
                <Box
                    display="flex"
                    alignItems="center"
                    justifyContent="space-between"
                    paddingLeft={2}
                    paddingRight={2}
                    paddingTop={1}
                    paddingBottom={1}
                >
                    {data.logLastUpdatedAt && <Typography color="textSecondary">
                        last updated <Timestamp timestamp={data.logLastUpdatedAt as string} />
                    </Typography>}
                    <Tooltip title={autoScroll ? 'Disable auto scroll' : 'Enable auto scroll'}>
                        <ToggleButton
                            size="small"
                            value="check"
                            selected={autoScroll}
                            onChange={() => setAutoScroll(!autoScroll)}
                        >
                            <AutoScrollIcon />
                        </ToggleButton>
                    </Tooltip>
                </Box>
            </Paper>
            {completed && loadedPercent < 100 && <LinearProgress variant="determinate" value={loadedPercent} />}
            <LogViewer
                logs={logs}
                loading={!completed}
                followOutput={autoScroll}
                scrollMode={props.scrollMode}
                sx={{
                    backgroundColor: darken(theme.palette.background.default, 0.5),
                    paddingTop: 1,
                    paddingBottom: 2,
                    paddingRight: 1,
                    minHeight: 120,
                    // ContainerLogViewer defaults to height:100%, which collapses outside a sized
                    // parent — an explicit height is what makes it a bounded scrolling box.
                    ...(props.height !== undefined && { height: props.height })
                }}
            />
        </Box>
    );
}

export default JobLogs;
