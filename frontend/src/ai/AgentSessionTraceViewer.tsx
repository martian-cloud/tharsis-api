import ChevronRightIcon from '@mui/icons-material/ChevronRight';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import Box from '@mui/material/Box';
import CircularProgress from '@mui/material/CircularProgress';
import Collapse from '@mui/material/Collapse';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import Typography from '@mui/material/Typography';
import { alpha, useTheme } from '@mui/material/styles';
import graphql from 'babel-plugin-relay/macro';
import { Suspense, useEffect, useMemo, useState, type FC, type ReactNode } from 'react';
import { PreloadedQuery, useLazyLoadQuery, usePreloadedQuery, useQueryLoader } from 'react-relay/hooks';
import type { AgentSessionTraceViewerRunsQuery } from './__generated__/AgentSessionTraceViewerRunsQuery.graphql';
import type { AgentSessionTraceViewerTraceQuery } from './__generated__/AgentSessionTraceViewerTraceQuery.graphql';

export const agentSessionRunsQuery = graphql`
    query AgentSessionTraceViewerRunsQuery($sessionId: String!) {
        node(id: $sessionId) {
            ... on AgentSession {
                runs(first: 100, sort: CREATED_AT_ASC) {
                    edges {
                        node {
                            id
                            status
                            metadata {
                                createdAt
                            }
                        }
                    }
                }
            }
        }
    }
`;

const AgentTraceQuery = graphql`
    query AgentSessionTraceViewerTraceQuery($input: AgentTraceInput!) {
        agentTrace(input: $input)
    }
`;

// --- Types matching gollem trace JSON ---

type FunctionCall = { id: string; name: string; arguments?: Record<string, any> };
type ToolSpec = { name: string; description?: string };
type LLMRequest = { system_prompt?: string; messages?: { role: string; content: string }[]; tools?: ToolSpec[] };
type LLMResponse = { texts?: string[]; function_calls?: FunctionCall[] };
type LLMCallData = { model?: string; input_tokens?: number; output_tokens?: number; request?: LLMRequest; response?: LLMResponse };
type ToolExecData = { tool_name: string; args?: Record<string, any>; result?: Record<string, any>; error?: string };
type EventData = { kind: string; data?: any };

type Span = {
    span_id: string;
    kind: string;
    name: string;
    duration: number;
    status: string;
    error?: string;
    children?: Span[];
    llm_call?: LLMCallData;
    tool_exec?: ToolExecData;
    event?: EventData;
};

// --- Helpers ---

function formatDuration(ns: number): string {
    if (ns < 1e6) return `${(ns / 1e3).toFixed(0)}µs`;
    if (ns < 1e9) return `${(ns / 1e6).toFixed(0)}ms`;
    return `${(ns / 1e9).toFixed(1)}s`;
}

const CodeBlock: FC<{ children: string; maxHeight?: number }> = ({ children, maxHeight = 300 }) => (
    <Box
        component="pre"
        sx={{
            m: 0, p: 1, fontSize: '0.7rem', fontFamily: 'monospace',
            bgcolor: 'action.hover', borderRadius: 1, overflow: 'auto',
            maxHeight, whiteSpace: 'pre-wrap', wordBreak: 'break-all',
        }}
    >
        {children}
    </Box>
);

const Section: FC<{ label: string; defaultOpen?: boolean; children: ReactNode }> = ({ label, defaultOpen = false, children }) => {
    const [open, setOpen] = useState(defaultOpen);
    return (
        <Box sx={{ my: 0.5 }}>
            <Typography
                variant="caption"
                onClick={() => setOpen(o => !o)}
                sx={{ cursor: 'pointer', fontWeight: 600, display: 'flex', alignItems: 'center', gap: 0.5, color: 'text.secondary' }}
            >
                {open ? <ExpandMoreIcon sx={{ fontSize: 12 }} /> : <ChevronRightIcon sx={{ fontSize: 12 }} />}
                {label}
            </Typography>
            <Collapse in={open}><Box sx={{ ml: 2, mt: 0.5 }}>{children}</Box></Collapse>
        </Box>
    );
};

const LLMCallDetail: FC<{ data: LLMCallData }> = ({ data }) => (
    <Box sx={{ ml: 2, mt: 0.5 }}>
        <Typography variant="caption" sx={{ color: 'text.secondary' }}>
            model: {data.model ?? '?'} · tokens: {data.input_tokens ?? '?'} in / {data.output_tokens ?? '?'} out
        </Typography>
        {data.request?.system_prompt && (
            <Section label="System Prompt">
                <CodeBlock>{data.request.system_prompt}</CodeBlock>
            </Section>
        )}
        {data.request?.messages && data.request.messages.length > 0 && (
            <Section label={`Messages (${data.request.messages.length})`}>
                {data.request.messages.map((msg, i) => (
                    <Box key={i} sx={{ mb: 0.5 }}>
                        <Typography variant="caption" sx={{ fontWeight: 600, color: msg.role === 'user' ? 'primary.main' : msg.role === 'assistant' ? 'success.main' : 'warning.main' }}>
                            {msg.role}
                        </Typography>
                        <CodeBlock maxHeight={150}>{msg.content}</CodeBlock>
                    </Box>
                ))}
            </Section>
        )}
        {data.request?.tools && data.request.tools.length > 0 && (
            <Section label={`Tools (${data.request.tools.length})`}>
                {data.request.tools.map((t, i) => (
                    <Typography key={i} variant="caption" component="div" sx={{ fontFamily: 'monospace' }}>
                        {t.name}{t.description ? ` — ${t.description}` : ''}
                    </Typography>
                ))}
            </Section>
        )}
        {data.response && (
            <Section label="Response" defaultOpen>
                {data.response.texts?.map((t, i) => (
                    <Box key={i} sx={{ mb: 0.5 }}>
                        <Typography variant="caption" sx={{ fontWeight: 600 }}>Text:</Typography>
                        <CodeBlock maxHeight={200}>{t}</CodeBlock>
                    </Box>
                ))}
                {data.response.function_calls?.map((fc, i) => (
                    <Box key={i} sx={{ mb: 0.5 }}>
                        <Typography variant="caption" sx={{ fontWeight: 600, color: 'warning.main' }}>
                            Tool Call: {fc.name} ({fc.id})
                        </Typography>
                        {fc.arguments && <CodeBlock>{JSON.stringify(fc.arguments, null, 2)}</CodeBlock>}
                    </Box>
                ))}
            </Section>
        )}
    </Box>
);

const ToolExecDetail: FC<{ data: ToolExecData }> = ({ data }) => (
    <Box sx={{ ml: 2, mt: 0.5 }}>
        {data.args && Object.keys(data.args).length > 0 && (
            <Section label="Arguments" defaultOpen>
                <CodeBlock>{JSON.stringify(data.args, null, 2)}</CodeBlock>
            </Section>
        )}
        {data.result && Object.keys(data.result).length > 0 && (
            <Section label="Result" defaultOpen>
                <CodeBlock>{JSON.stringify(data.result, null, 2)}</CodeBlock>
            </Section>
        )}
        {data.error && (
            <Typography variant="caption" sx={{ color: 'error.main', display: 'block', fontFamily: 'monospace', ml: 0.5 }}>
                Error: {data.error}
            </Typography>
        )}
    </Box>
);

const SpanNode: FC<{ span: Span; depth?: number }> = ({ span, depth = 0 }) => {
    const theme = useTheme();
    const [open, setOpen] = useState(depth < 2);
    const hasChildren = span.children && span.children.length > 0;
    const hasDetail = !!span.llm_call || !!span.tool_exec || !!span.event;
    const expandable = hasChildren || hasDetail;

    const kindColors: Record<string, string> = {
        agent_execute: theme.palette.primary.main,
        llm_call: theme.palette.info.main,
        tool_exec: theme.palette.warning.main,
        event: theme.palette.secondary.main,
    };

    return (
        <Box sx={{ ml: depth > 0 ? 2 : 0, borderLeft: depth > 0 ? `1px solid ${alpha(theme.palette.divider, 0.5)}` : 'none', pl: depth > 0 ? 1 : 0 }}>
            <Box
                onClick={() => expandable && setOpen(o => !o)}
                sx={{
                    display: 'flex', alignItems: 'center', gap: 0.5, py: 0.3, px: 0.5,
                    cursor: expandable ? 'pointer' : 'default',
                    '&:hover': expandable ? { bgcolor: 'action.hover' } : {},
                    borderRadius: 1,
                }}
            >
                {expandable ? (
                    open ? <ExpandMoreIcon sx={{ fontSize: 14 }} /> : <ChevronRightIcon sx={{ fontSize: 14 }} />
                ) : (
                    <Box sx={{ width: 14 }} />
                )}
                <Box
                    sx={{
                        px: 0.5, borderRadius: 0.5, fontSize: '0.65rem', fontWeight: 700,
                        color: 'white', bgcolor: kindColors[span.kind] ?? theme.palette.text.secondary,
                        lineHeight: 1.6, textTransform: 'uppercase', letterSpacing: 0.5,
                    }}
                >
                    {span.kind.replace('_', ' ')}
                </Box>
                <Typography variant="caption" sx={{ flex: 1, fontFamily: 'monospace', fontWeight: 500 }}>
                    {span.name}
                </Typography>
                <Typography variant="caption" sx={{ color: 'text.secondary', fontFamily: 'monospace' }}>
                    {formatDuration(span.duration)}
                </Typography>
                {span.status === 'error' && (
                    <Typography variant="caption" sx={{ color: 'error.main', fontWeight: 700 }}>ERR</Typography>
                )}
            </Box>
            {span.error && (
                <Typography variant="caption" sx={{ ml: 3, color: 'error.main', display: 'block', fontFamily: 'monospace' }}>
                    {span.error}
                </Typography>
            )}
            <Collapse in={open}>
                {span.llm_call && <LLMCallDetail data={span.llm_call} />}
                {span.tool_exec && <ToolExecDetail data={span.tool_exec} />}
                {span.event && (
                    <Box sx={{ ml: 2, mt: 0.5 }}>
                        <Section label={`Event: ${span.event.kind}`} defaultOpen>
                            <CodeBlock>{JSON.stringify(span.event.data, null, 2)}</CodeBlock>
                        </Section>
                    </Box>
                )}
                {span.children?.map(child => (
                    <SpanNode key={child.span_id} span={child} depth={depth + 1} />
                ))}
            </Collapse>
        </Box>
    );
};

// --- Trace content for a single run (uses Relay lazy load query) ---

function TraceContent({ queryRef }: { queryRef: PreloadedQuery<AgentSessionTraceViewerTraceQuery> }) {
    const data = usePreloadedQuery<AgentSessionTraceViewerTraceQuery>(AgentTraceQuery, queryRef);

    const trace = useMemo(() => {
        if (!data?.agentTrace) return null;
        try {
            return JSON.parse(data.agentTrace);
        } catch {
            return data.agentTrace;
        }
    }, [data]);

    if (!trace) {
        return (
            <Typography color="text.secondary" sx={{ py: 2, textAlign: 'center' }}>
                No trace data available.
            </Typography>
        );
    }

    return (
        <Box>
            <Box sx={{ display: 'flex', gap: 2, mb: 1, px: 1 }}>
                <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                    Trace: {trace.trace_id}
                </Typography>
                {trace.metadata?.model && (
                    <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                        Model: {trace.metadata.model}
                    </Typography>
                )}
            </Box>
            {trace.root_span && <SpanNode span={trace.root_span} />}
        </Box>
    );
}

const terminalStatuses = new Set(['finished', 'errored', 'cancelled']);

function AgentSessionTraceViewer({ sessionId, fetchKey = 0 }: { sessionId: string; fetchKey?: number }) {
    const data = useLazyLoadQuery<AgentSessionTraceViewerRunsQuery>(agentSessionRunsQuery, { sessionId }, { fetchPolicy: 'store-and-network', fetchKey });
    const runs: readonly { id: string; status: string; metadata: { createdAt: string } }[] =
        data?.node?.runs?.edges?.map(e => e!.node!).filter(Boolean) ?? [];

    const [selectedRunId, setSelectedRunId] = useState<string | null>(null);
    const [traceQueryRef, loadTraceQuery] = useQueryLoader<AgentSessionTraceViewerTraceQuery>(AgentTraceQuery);

    useEffect(() => {
        if (runs.length > 0 && !selectedRunId) {
            setSelectedRunId(runs[runs.length - 1].id);
        }
    }, [runs.length]);

    const activeRun = runs.find(r => r.id === selectedRunId);
    const canShowTrace = activeRun && terminalStatuses.has(activeRun.status);

    useEffect(() => {
        if (selectedRunId && canShowTrace) {
            loadTraceQuery({ input: { runId: selectedRunId } }, { fetchPolicy: 'store-and-network' });
        }
    }, [selectedRunId, canShowTrace, loadTraceQuery]);

    if (!selectedRunId) {
        return (
            <Typography color="text.secondary" sx={{ py: 2, textAlign: 'center' }}>
                No runs yet.
            </Typography>
        );
    }

    return (
        <>
            {runs.length > 1 && (
                <Select
                    size="small"
                    value={selectedRunId}
                    onChange={e => setSelectedRunId(e.target.value)}
                    sx={{ minWidth: 200, fontSize: '0.8rem' }}
                >
                    {runs.map((run, i: number) => (
                        <MenuItem key={run.id} value={run.id}>
                            Run {i + 1} — {run.status}
                        </MenuItem>
                    ))}
                </Select>
            )}
            {!canShowTrace ? (
                <Typography color="text.secondary" sx={{ py: 2, textAlign: 'center' }}>
                    Run is still in progress. Refresh to view trace when the run completes.
                </Typography>
            ) : traceQueryRef ? (
                <Suspense fallback={<Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}><CircularProgress /></Box>}>
                    <TraceContent queryRef={traceQueryRef} />
                </Suspense>
            ) : null}
        </>
    );
}

export default AgentSessionTraceViewer;
