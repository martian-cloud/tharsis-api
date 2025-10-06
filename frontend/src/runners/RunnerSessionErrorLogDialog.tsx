import CloseIcon from '@mui/icons-material/Close';
import { Box, Button, CircularProgress, Dialog, DialogActions, DialogContent, DialogTitle, Paper, Typography, darken } from "@mui/material";
import IconButton from '@mui/material/IconButton';
import { useTheme } from '@mui/material/styles';
import useMediaQuery from '@mui/material/useMediaQuery';
import graphql from 'babel-plugin-relay/macro';
import moment from 'moment';
import { Suspense, useEffect, useState } from "react";
import { fetchQuery, useLazyLoadQuery, useRelayEnvironment } from 'react-relay/hooks';
import LogViewer from '../workspace/runs/LogViewer';
import { RunnerSessionErrorLogDialogQuery } from "./__generated__/RunnerSessionErrorLogDialogQuery.graphql";

const query = graphql`
    query  RunnerSessionErrorLogDialogQuery($id: String!, $startOffset: Int!, $limit: Int!) {
        node(id: $id) {
            ...on RunnerSession {
                id
                errorCount
                errorLog {
                    lastUpdatedAt
                    size
                    data(startOffset:$startOffset, limit:$limit)
                }
            }
        }
    }
`;

const bytes = (str: string) => {
    const size = new Blob([str]).size;
    return size;
}

const LOG_CHUNK_SIZE_BYTES = 1024 * 1024;

interface Props {
    sessionId: string
    onClose: (confirm?: boolean) => void
}

function RunnerSessionErrorLogDialog(props: Props) {
    const { sessionId, onClose } = props;
    const theme = useTheme();
    const fullScreen = useMediaQuery(theme.breakpoints.down('md'));

    return (
        <Dialog
            open
            maxWidth="lg"
            fullWidth
            fullScreen={fullScreen}
        >
            <DialogTitle sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                Runner Session Errors
                <IconButton
                    color="inherit"
                    size="small"
                    onClick={() => onClose()}
                >
                    <CloseIcon />
                </IconButton>
            </DialogTitle>
            <DialogContent dividers sx={{ flex: 1, padding: 0, minHeight: 600, display: 'flex', flexDirection: 'column' }}>
                <Suspense fallback={<Box
                    sx={{
                        position: 'absolute',
                        top: 0,
                        left: 0,
                        width: '100%',
                        minHeight: '100%',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center'
                    }}>
                    <CircularProgress />
                </Box>}>
                    <ErrorLogDialogContent sessionId={sessionId} />
                </Suspense>
            </DialogContent>
            {!fullScreen && <DialogActions>
                <Button color="inherit" onClick={() => onClose()}>
                    Close
                </Button>
            </DialogActions>}
        </Dialog>
    );
}

interface ErrorLogDialogContentProps {
    sessionId: string
}

function ErrorLogDialogContent(props: ErrorLogDialogContentProps) {
    const queryData = useLazyLoadQuery<RunnerSessionErrorLogDialogQuery>(query, { id: props.sessionId, startOffset: 0, limit: 51200 }, { fetchPolicy: 'network-only' });

    const theme = useTheme();
    const [errorCount, setErrorCount] = useState(0);
    const [logs, setLogs] = useState('');
    const [currentLogSize, setCurrentLogSize] = useState(0);
    const [actualLogSize, setActualLogSize] = useState(0);
    const [lastUpdatedAt, setLastUpdatedAt] = useState('');
    const [loading, setLoading] = useState<boolean>(false);
    const environment = useRelayEnvironment();

    useEffect(() => {
        const errorLog = queryData.node?.errorLog;
        if (errorLog) {
            const data = errorLog.data || '';
            setLogs(data);
            setCurrentLogSize(bytes(data));
            setActualLogSize(errorLog.size);
            setLastUpdatedAt(errorLog.lastUpdatedAt);
        }

        setErrorCount(queryData.node?.errorCount || 0);
    }, [props.sessionId]);

    useEffect(() => {
        if (loading || currentLogSize >= actualLogSize) {
            return;
        }

        setLoading(true);

        fetchQuery<RunnerSessionErrorLogDialogQuery>(
            environment,
            query,
            { id: props.sessionId, startOffset: currentLogSize, limit: LOG_CHUNK_SIZE_BYTES },
            { fetchPolicy: 'network-only' }
        ).toPromise().then(async response => {
            setLoading(false);
            const errorLog = response?.node?.errorLog;
            if (errorLog) {
                setLogs(logs + errorLog.data);
                setActualLogSize(errorLog.size);
                setCurrentLogSize(prev => prev + bytes(errorLog.data));
                setLastUpdatedAt(errorLog.lastUpdatedAt);
            }
            setErrorCount(response?.node?.errorCount || 0);
        });
    }, [props.sessionId, actualLogSize, currentLogSize, logs, loading, environment]);

    return (
        <Box display="flex" flexDirection="column" flex={1}>
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
                    <Typography>
                        {errorCount} error{errorCount === 1 ? '' : 's'}
                    </Typography>
                    {lastUpdatedAt && <Typography color="textSecondary">
                        last updated {moment(lastUpdatedAt as moment.MomentInput).fromNow()}
                    </Typography>}
                </Box>
            </Paper>
            <Box display="flex" flexDirection="column" flex={1} position="relative">
                <Box position="absolute" top={0} left={0} width="100%" height="100%">
                    <LogViewer
                        logs={logs}
                        sx={{
                            backgroundColor: darken(theme.palette.background.default, 0.5),
                            paddingTop: 1,
                            paddingBottom: 1,
                            paddingRight: 1,
                            minHeight: 120
                        }}
                    />
                </Box>
            </Box>
        </Box>
    );
}

export default RunnerSessionErrorLogDialog;
