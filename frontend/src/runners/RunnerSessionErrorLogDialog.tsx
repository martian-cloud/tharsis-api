import CloseIcon from '@mui/icons-material/Close';
import { Box, Button, CircularProgress, Dialog, DialogActions, DialogContent, DialogTitle, Paper, Typography, darken } from "@mui/material";
import IconButton from '@mui/material/IconButton';
import { useTheme } from '@mui/material/styles';
import useMediaQuery from '@mui/material/useMediaQuery';
import graphql from 'babel-plugin-relay/macro';
import { useEffect, useState } from "react";
import { fetchQuery, useRelayEnvironment } from 'react-relay/hooks';
import Timestamp from '../common/Timestamp';
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
                <ErrorLogDialogContent sessionId={sessionId} />
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
    const theme = useTheme();
    const environment = useRelayEnvironment();

    const [errorCount, setErrorCount] = useState(0);
    const [logs, setLogs] = useState<string | null>(null);
    const [currentLogSize, setCurrentLogSize] = useState(0);
    const [actualLogSize, setActualLogSize] = useState(0);
    const [lastUpdatedAt, setLastUpdatedAt] = useState('');
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        if (logs !== null && (loading || currentLogSize >= actualLogSize)) {
            return;
        }

        const isInitial = logs === null;
        const startOffset = isInitial ? 0 : currentLogSize;
        const limit = isInitial ? 51200 : LOG_CHUNK_SIZE_BYTES;

        if (!isInitial) setLoading(true);

        let cancelled = false;
        fetchQuery<RunnerSessionErrorLogDialogQuery>(
            environment,
            query,
            { id: props.sessionId, startOffset, limit },
            { fetchPolicy: 'network-only' }
        ).toPromise().then(response => {
            if (cancelled) return;
            const errorLog = response?.node?.errorLog;
            if (isInitial) {
                const data = errorLog?.data ?? '';
                setLogs(data);
                setCurrentLogSize(bytes(data));
                setActualLogSize(errorLog?.size ?? 0);
                setLastUpdatedAt(errorLog?.lastUpdatedAt ?? '');
            } else {
                setLoading(false);
                if (errorLog) {
                    setLogs(prev => (prev ?? '') + errorLog.data);
                    setActualLogSize(errorLog.size);
                    setCurrentLogSize(prev => prev + bytes(errorLog.data));
                    setLastUpdatedAt(errorLog.lastUpdatedAt);
                }
            }
            setErrorCount(response?.node?.errorCount ?? 0);
        });
        return () => { cancelled = true; };
    }, [props.sessionId, environment, logs, currentLogSize, actualLogSize, loading]);

    if (logs === null) {
        return (
            <Box
                sx={{
                    position: 'absolute',
                    top: 0,
                    left: 0,
                    width: '100%',
                    minHeight: '100%',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center'
                }}
            >
                <CircularProgress />
            </Box>
        );
    }

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
                        last updated <Timestamp timestamp={lastUpdatedAt} />
                    </Typography>}
                </Box>
            </Paper>
            <Box display="flex" flexDirection="column" flex={1} position="relative">
                <Box position="absolute" top={0} left={0} width="100%" height="100%">
                    <LogViewer
                        logs={logs}
                        scrollMode="container"
                        disableDeepLink
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
