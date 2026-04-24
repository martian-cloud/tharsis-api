import {
    AuiIf,
    ComposerPrimitive,
    MessagePartPrimitive,
    MessagePrimitive,
    SuggestionPrimitive,
    ThreadPrimitive,
} from '@assistant-ui/react';
import BuildIcon from '@mui/icons-material/Build';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import ChevronRightIcon from '@mui/icons-material/ChevronRight';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import SendIcon from '@mui/icons-material/Send';
import StopIcon from '@mui/icons-material/Stop';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';
import Collapse from '@mui/material/Collapse';
import IconButton from '@mui/material/IconButton';
import Typography from '@mui/material/Typography';
import { alpha } from '@mui/material/styles';
import { useMemo, useState, type FC } from 'react';
import LoadingDots from '../common/LoadingDots';
import Markdown from '../common/Markdown';

const ToolFallback: FC<{ toolName: string; argsText?: string; result?: unknown; status?: { type: string; reason?: string }; showToolCalls?: boolean }> = ({ toolName, argsText, result, status, showToolCalls = true }) => {
    const [open, setOpen] = useState(false);
    const isRunning = status?.type === 'running';

    if (!showToolCalls) return null;

    return (
        <Box sx={{ border: 1, borderColor: 'divider', borderRadius: 2, my: 1, overflow: 'hidden', bgcolor: 'rgb(29, 31, 33)' }}>
            <Box
                onClick={() => setOpen(o => !o)}
                sx={{ display: 'flex', alignItems: 'center', gap: 1, px: 1.5, py: 1, cursor: 'pointer', '&:hover': { bgcolor: 'action.hover' } }}
            >
                {isRunning ? (
                    <CircularProgress size={16} />
                ) : status?.type === 'incomplete' ? (
                    <ErrorOutlineIcon sx={{ fontSize: 16, color: 'error.main' }} />
                ) : (
                    <CheckCircleOutlineIcon sx={{ fontSize: 16, color: 'success.main' }} />
                )}
                <BuildIcon sx={{ fontSize: 14, color: 'text.secondary' }} />
                <Typography variant="caption" sx={{ fontWeight: 600, flex: 1 }}>
                    {isRunning ? 'Calling' : 'Used'}: {toolName}
                </Typography>
                {open ? <ExpandMoreIcon sx={{ fontSize: 16 }} /> : <ChevronRightIcon sx={{ fontSize: 16 }} />}
            </Box>
            <Collapse in={open}>
                <Box sx={{ px: 1.5, pb: 1.5, fontSize: '0.75rem', overflowX: 'auto' }}>
                    {argsText && (
                        <Box component="pre" sx={{ m: 0, whiteSpace: 'pre-wrap', color: 'text.secondary', fontFamily: 'monospace', fontSize: 'inherit' }}>
                            {argsText}
                        </Box>
                    )}
                    {result !== undefined && (
                        <Box sx={{ mt: 1, pt: 1, borderTop: 1, borderColor: 'divider' }}>
                            <Typography variant="caption" sx={{ fontWeight: 600 }}>Result:</Typography>
                            <Box component="pre" sx={{ m: 0, whiteSpace: 'pre-wrap', color: 'text.secondary', fontFamily: 'monospace', fontSize: 'inherit' }}>
                                {typeof result === 'string' ? result : JSON.stringify(result, null, 2)}
                            </Box>
                        </Box>
                    )}
                </Box>
            </Collapse>
        </Box>
    );
};

const UserMessage: FC = () => (
    <MessagePrimitive.Root>
        <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 0.5, px: 1, py: 0.5 }}>
            <ChevronRightIcon sx={{ fontSize: 18, color: 'primary.main', mt: '2px' }} />
            <Typography variant="body2" component="div" sx={{ '& p': { m: 0 }, fontFamily: 'monospace', fontWeight: 700, color: 'primary.main', overflowWrap: 'anywhere' }}>
                <MessagePrimitive.Content />
            </Typography>
        </Box>
    </MessagePrimitive.Root>
);

const AssistantMessage: FC<{ showToolCalls?: boolean }> = ({ showToolCalls = true }) => {
    return (
        <MessagePrimitive.Root>
            <Box sx={{ px: 1, py: 0.5, maxWidth: '90%' }}>
                <MessagePrimitive.Parts
                    components={{
                        Text: ({ text }) => text ? (
                            <Box sx={{ my: 0.5, overflowWrap: 'anywhere', '& code': { whiteSpace: 'pre-wrap' } }}>
                                <Markdown>{text}</Markdown>
                                <MessagePartPrimitive.InProgress>
                                    <LoadingDots />
                                </MessagePartPrimitive.InProgress>
                            </Box>
                        ) : (
                            <MessagePartPrimitive.InProgress>
                                <LoadingDots />
                            </MessagePartPrimitive.InProgress>
                        ),
                        tools: {
                            Fallback: (props) => <ToolFallback {...props} showToolCalls={showToolCalls} />,
                        },
                    }}
                />
            </Box>
        </MessagePrimitive.Root>
    );
};

const SuggestionCard = () => (
    <SuggestionPrimitive.Trigger send asChild>
        <Button variant="outlined" size="small" sx={{ textTransform: 'none', justifyContent: 'flex-start', borderRadius: 2 }}>
            <SuggestionPrimitive.Title />
        </Button>
    </SuggestionPrimitive.Trigger>
);

function AgentSessionChat({ showToolCalls = true }: { showToolCalls?: boolean }) {
    const BoundAssistantMessage = useMemo(() => () => <AssistantMessage showToolCalls={showToolCalls} />, [showToolCalls]);

    return (
        <ThreadPrimitive.Root style={{ display: 'flex', flexDirection: 'column', minHeight: 0, height: '100%' }}>

            <ThreadPrimitive.Viewport autoScroll style={{ flex: 1, overflowY: 'auto', display: 'flex', flexDirection: 'column', minHeight: 0 }}>
                <ThreadPrimitive.Empty>
                    <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: 1, opacity: 0.6 }}>
                        <Typography variant="h6" color="text.secondary">How can I help you?</Typography>
                    </Box>
                    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1, px: 2, pb: 2 }}>
                        <ThreadPrimitive.Suggestions>
                            {() => <SuggestionCard />}
                        </ThreadPrimitive.Suggestions>
                    </Box>
                </ThreadPrimitive.Empty>
                <ThreadPrimitive.Messages components={{ UserMessage, AssistantMessage: BoundAssistantMessage }} />
            </ThreadPrimitive.Viewport>

            <Box sx={{
                p: 1,
                m: 1.5,
                border: 1,
                borderColor: 'divider',
                borderRadius: 3,
                '&:focus-within': { borderColor: 'transparent', boxShadow: (t: any) => `0 0 8px 2px ${alpha(t.palette.primary.main, 0.7)}` }
            }}>
                <ComposerPrimitive.Root style={{ display: 'flex', alignItems: 'flex-end' }}>
                    <ComposerPrimitive.Input
                        placeholder="Ask something…"
                        autoFocus
                        minRows={4}
                        maxRows={4}
                        style={{
                            flex: 1, border: 'none', outline: 'none', resize: 'none',
                            fontFamily: 'inherit', fontSize: '0.875rem', padding: '8px 12px',
                            background: 'transparent', color: 'inherit',
                        }}
                    />
                    <AuiIf condition={s => !s.thread.isRunning}>
                        <ComposerPrimitive.Send asChild>
                            <IconButton size="small" color="primary">
                                <SendIcon fontSize="small" />
                            </IconButton>
                        </ComposerPrimitive.Send>
                    </AuiIf>
                    <AuiIf condition={s => s.thread.isRunning}>
                        <ComposerPrimitive.Cancel asChild>
                            <IconButton size="small" color="info">
                                <StopIcon fontSize="small" />
                            </IconButton>
                        </ComposerPrimitive.Cancel>
                    </AuiIf>
                </ComposerPrimitive.Root>
            </Box>
        </ThreadPrimitive.Root>
    );
}

export default AgentSessionChat;
