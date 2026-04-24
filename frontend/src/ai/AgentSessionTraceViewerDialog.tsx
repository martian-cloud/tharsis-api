import BugReportIcon from '@mui/icons-material/BugReport';
import CloseIcon from '@mui/icons-material/Close';
import RefreshIcon from '@mui/icons-material/Refresh';
import Box from '@mui/material/Box';
import CircularProgress from '@mui/material/CircularProgress';
import Dialog from '@mui/material/Dialog';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import IconButton from '@mui/material/IconButton';
import { Suspense, useState } from 'react';
import AgentSessionTraceViewer from './AgentSessionTraceViewer';

type AgentSessionTraceViewerDialogProps = {
    sessionId: string;
    onClose: () => void;
};

export function AgentSessionTraceViewerDialog({ sessionId, onClose }: AgentSessionTraceViewerDialogProps) {
    const [fetchKey, setFetchKey] = useState(0);

    return (
        <Dialog open onClose={onClose} maxWidth="md" fullWidth>
            <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1, py: 1 }}>
                <BugReportIcon fontSize="small" />
                <Box component="span" sx={{ flex: 1 }}>Agent Trace</Box>
                <IconButton size="small" onClick={() => setFetchKey(k => k + 1)} title="Refresh">
                    <RefreshIcon fontSize="small" />
                </IconButton>
                <IconButton size="small" onClick={onClose}>
                    <CloseIcon fontSize="small" />
                </IconButton>
            </DialogTitle>
            <DialogContent sx={{ p: 1 }}>
                <Suspense fallback={<Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}><CircularProgress /></Box>}>
                    <AgentSessionTraceViewer sessionId={sessionId} fetchKey={fetchKey} />
                </Suspense>
            </DialogContent>
        </Dialog>
    );
}

export function AgentSessionTraceViewerButton({ sessionId }: { sessionId: string }) {
    const [open, setOpen] = useState(false);

    return (
        <>
            <IconButton size="small" onClick={() => setOpen(true)} title="View traces">
                <BugReportIcon fontSize="small" />
            </IconButton>
            {open && (
                <AgentSessionTraceViewerDialog
                    sessionId={sessionId}
                    onClose={() => setOpen(false)}
                />
            )}
        </>
    );
}
