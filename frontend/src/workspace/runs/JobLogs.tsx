import AutoScrollIcon from '@mui/icons-material/ArrowCircleDown';
import { Box, darken, LinearProgress, Paper, ToggleButton, Tooltip, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useMemo, useRef, useState } from 'react';
import { useFragment, useSubscription } from 'react-relay/hooks';
import { GraphQLSubscriptionConfig, RecordSourceProxy } from 'relay-runtime';
import Timestamp from '../../common/Timestamp';
import LogViewer from './LogViewer';
import { JobLogsFragment_logs$key } from './__generated__/JobLogsFragment_logs.graphql';
import { JobLogsSubscription, JobLogsSubscription$data } from './__generated__/JobLogsSubscription.graphql';

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
    fragmentRef: JobLogsFragment_logs$key
    enableAutoScrollByDefault?: boolean
}

const bytes = (str: string) => {
    const size = new Blob([str]).size;
    return size;
}

function JobLogs(props: Props) {
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
            logs(startOffset:0, limit:51200)
        }
      `, props.fragmentRef)

    const [logs, setLogs] = useState(data.logs);
    const [currentLogSize, setCurrentLogSize] = useState(bytes(data.logs));
    const [actualLogSize, setActualLogSize] = useState(data.logSize);
    const [autoScroll, setAutoScroll] = useState(props.enableAutoScrollByDefault);
    const [completed, setCompleted] = useState(data.completed);
    // Tracks the byte size already appended so we can dedupe events without re-measuring
    // the whole accumulated buffer on every event.
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
                sx={{
                    backgroundColor: darken(theme.palette.background.default, 0.5),
                    paddingTop: 1,
                    paddingBottom: 2,
                    paddingRight: 1,
                    minHeight: 120
                }}
            />
        </Box>
    );
}

export default JobLogs;
