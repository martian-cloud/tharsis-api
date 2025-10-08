import AutoScrollIcon from '@mui/icons-material/ArrowCircleDown';
import { Box, darken, LinearProgress, Paper, ToggleButton, Tooltip, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import moment from 'moment';
import { useEffect, useMemo, useState } from 'react';
import { useFragment, useRelayEnvironment, useSubscription } from 'react-relay/hooks';
import { fetchQuery, GraphQLSubscriptionConfig, RecordSourceProxy } from 'relay-runtime';
import LogViewer from './LogViewer';
import { JobLogsFragment_logs$key } from './__generated__/JobLogsFragment_logs.graphql';
import { JobLogsQuery } from './__generated__/JobLogsQuery.graphql';
import { JobLogsSubscription, JobLogsSubscription$data } from './__generated__/JobLogsSubscription.graphql';

const subscription = graphql`subscription JobLogsSubscription($input: JobLogStreamSubscriptionInput!) {
    jobLogStreamEvents(input: $input) {
      size
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

const LOG_CHUNK_SIZE_BYTES = 1024 * 1024;

function JobLogs(props: Props) {
    const theme = useTheme();
    const data = useFragment<JobLogsFragment_logs$key>(
        graphql`
        fragment JobLogsFragment_logs on Job
        {
            id
            status
            logLastUpdatedAt
            logSize
            logs(startOffset:0, limit:51200)
        }
      `, props.fragmentRef)

    const [logs, setLogs] = useState(data.logs);
    const [currentLogSize, setCurrentLogSize] = useState(bytes(data.logs));
    const [actualLogSize, setActualLogSize] = useState(data.logSize);
    const [lastLogEventSize, setLastLogEventSize] = useState(data.logSize);
    const [loading, setLoading] = useState<boolean>(false);
    const [autoScroll, setAutoScroll] = useState(props.enableAutoScrollByDefault);
    const environment = useRelayEnvironment();

    const config = useMemo<GraphQLSubscriptionConfig<JobLogsSubscription>>(() => ({
        variables: { input: { jobId: data.id, lastSeenLogSize: data.logSize } },
        subscription,
        onCompleted: () => console.log("Subscription completed"),
        onError: () => console.warn("Subscription error"),
        updater: (store: RecordSourceProxy, payload: JobLogsSubscription$data | null | undefined) => {
            if (payload) {
                setLastLogEventSize(payload.jobLogStreamEvents.size);
            }
        }
    }), [data.id]);
    useSubscription<JobLogsSubscription>(config);

    useEffect(() => {
        if (lastLogEventSize > actualLogSize) {
            setActualLogSize(lastLogEventSize);
        }
    }, [lastLogEventSize, actualLogSize]);

    useEffect(() => {
        if (loading || currentLogSize >= actualLogSize) {
            return;
        }

        setLoading(true);

        fetchQuery<JobLogsQuery>(
            environment,
            graphql`
              query JobLogsQuery($id: String!, $startOffset: Int!, $limit: Int!) {
                job(id: $id) {
                  id
                  logLastUpdatedAt
                  logSize
                  logs(startOffset:$startOffset, limit:$limit)
                }
              }
            `,
            { id: data.id, startOffset: currentLogSize, limit: LOG_CHUNK_SIZE_BYTES },
            { fetchPolicy: 'network-only' }
        ).toPromise().then(async response => {
            setLoading(false);
            const job = response?.job;
            if (job) {
                setLogs(logs + job.logs);
                setActualLogSize(job.logSize);
                setCurrentLogSize(prev => prev + bytes(job.logs));
            }
        });
    }, [data, actualLogSize, currentLogSize, logs, loading, environment]);

    useEffect(() => {
        if (autoScroll) {
            // Use timeout here to account for any dom updates when a status change occurs
            // to ensure that the scroll is done after the dom is updated
            const timeoutId = setTimeout(() => {
                scrollToBottom();
            }, 200);

            // Cleanup function
            return () => clearTimeout(timeoutId);
        }
    }, [logs, autoScroll]);

    const loadedPercent = useMemo(() => (currentLogSize / actualLogSize) * 100, [currentLogSize, actualLogSize]);

    const scrollToBottom = () => {
        window.scrollTo(0, document.body.scrollHeight);
    };

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
                        last updated {moment(data.logLastUpdatedAt as moment.MomentInput).fromNow()}
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
            {data.status === 'finished' && loadedPercent < 100 && <LinearProgress variant="determinate" value={loadedPercent} />}
            <LogViewer
                logs={logs}
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
