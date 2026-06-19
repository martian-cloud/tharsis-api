import KeyboardArrowDownIcon from '@mui/icons-material/KeyboardArrowDown';
import KeyboardArrowUpIcon from '@mui/icons-material/KeyboardArrowUp';
import {
    Alert,
    Box,
    Checkbox,
    Chip,
    CircularProgress,
    Collapse,
    FormControl,
    InputLabel,
    List,
    ListItemText,
    MenuItem,
    Paper,
    Select,
    SelectChangeEvent,
    Switch,
    Table,
    TableBody,
    TableCell,
    TableContainer,
    TableHead,
    TableRow,
    Typography,
    useMediaQuery,
    useTheme,
} from '@mui/material';
import { alpha } from '@mui/material/styles';
import graphql from 'babel-plugin-relay/macro';
import throttle from 'lodash.throttle';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { atomDark as prismTheme } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { ChangeEvent, Fragment, KeyboardEvent, Suspense, memo, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { fetchQuery, useLazyLoadQuery, useRelayEnvironment, useSubscription } from 'react-relay/hooks';
import { useSearchParams } from 'react-router-dom';
import { GraphQLSubscriptionConfig } from 'relay-runtime';
import CopyButton from '../../common/CopyButton';
import SearchInput from '../../common/SearchInput';
import Timestamp from '../../common/Timestamp';
import AdminAreaBreadcrumbs from '../AdminAreaBreadcrumbs';
import { AdminAreaLogsQuery, AdminLogTailLevel } from './__generated__/AdminAreaLogsQuery.graphql';
import { AdminAreaLogsSubscription, AdminLogTailSubscriptionInput } from './__generated__/AdminAreaLogsSubscription.graphql';

const INITIAL_ITEM_COUNT = 100;
const EXPAND_MAX_HEIGHT = 400; // px cap on expanded fields/message/stack before it scrolls
const MAX_PENDING_ENTRIES = 1000; // cap on buffered live entries awaiting the next flush
const MAX_LIVE_ENTRIES = 500; // cap on live entries retained in view
const LIVE_FLUSH_INTERVAL_MS = 250; // how often buffered live entries are flushed to the table

const DESCRIPTION = 'Recent API server logs from a capped buffer — handy for quick debugging; complete history lives in the central logging system.';

const LOG_LEVELS: AdminLogTailLevel[] = ['DEBUG', 'INFO', 'WARN', 'ERROR'];

const BREADCRUMB_ROUTES = [{ title: 'logs', path: 'logs' }];

const logsQuery = graphql`
    query AdminAreaLogsQuery($limit: Int, $levels: [AdminLogTailLevel!], $search: String) {
        config {
            adminLogTailStorePluginType
        }
        adminLogTail(limit: $limit, levels: $levels, search: $search) {
            id
            timestamp
            level
            message
            caller
            stack
            fields
        }
    }
`;

const logsSubscription = graphql`
    subscription AdminAreaLogsSubscription($input: AdminLogTailSubscriptionInput!) {
        adminLogTailEvents(input: $input) {
            logEntry {
                id
                timestamp
                level
                message
                caller
                stack
                fields
            }
            error
        }
    }
`;

interface LogEntry {
    id: string;
    timestamp: string;
    level: string;
    message: string;
    caller: string | null;
    stack: string | null;
    fields: string | null;
}

function levelColor(level: string): 'default' | 'success' | 'info' | 'warning' | 'error' {
    switch (level.toLowerCase()) {
        case 'debug': return 'default';
        case 'info': return 'info';
        case 'warn': case 'warning': return 'warning';
        case 'error': case 'fatal': case 'dpanic': case 'panic': return 'error';
        default: return 'default';
    }
}

// matchesFilter mirrors the server's Matches; guards the live stream during resubscribes.
// Empty levels matches all.
function matchesFilter(entry: LogEntry, levels: AdminLogTailLevel[], search: string): boolean {
    if (levels.length > 0 && !levels.some(l => l.toLowerCase() === entry.level.toLowerCase())) {
        return false;
    }
    if (!search) {
        return true;
    }
    const needle = search.toLowerCase();
    return entry.message.toLowerCase().includes(needle)
        || (entry.caller?.toLowerCase().includes(needle) ?? false)
        || (entry.fields?.toLowerCase().includes(needle) ?? false);
}

interface LiveTailProps {
    levels: AdminLogTailLevel[];
    search: string;
    onEntry: (entry: LogEntry) => void;
    onError: (error: Error) => void;
}

function LiveTailSubscriber({ levels, search, onEntry, onError }: LiveTailProps) {
    const config = useMemo<GraphQLSubscriptionConfig<AdminAreaLogsSubscription>>(
        () => {
            const input: AdminLogTailSubscriptionInput = {};
            if (levels.length > 0) {
                input.levels = levels;
            }
            if (search) {
                input.search = search;
            }
            return {
                subscription: logsSubscription,
                variables: { input },
                onError,
                onNext: (payload: AdminAreaLogsSubscription['response'] | null | undefined) => {
                    const event = payload?.adminLogTailEvents;
                    if (!event) {
                        return;
                    }
                    // A terminal stream error (e.g. the backend log connection dropped) ends the tail.
                    if (event.error) {
                        onError(new Error(event.error));
                        return;
                    }
                    const e = event.logEntry;
                    if (e) {
                        onEntry({
                            id: e.id,
                            timestamp: e.timestamp as string,
                            level: e.level,
                            message: e.message,
                            caller: e.caller ?? null,
                            stack: e.stack ?? null,
                            fields: e.fields ?? null,
                        });
                    }
                },
            };
        },
        [levels, search, onEntry, onError]
    );

    useSubscription<AdminAreaLogsSubscription>(config);
    return null;
}

interface LogEntryRowProps {
    entry: LogEntry;
    isExpanded: boolean;
    onToggle: (id: string) => void;
    mobile: boolean;
    isNew: boolean;
}

// width:0 + minWidth:100% keeps wide content from resizing the table columns.
const detailBlock = {
    maxHeight: EXPAND_MAX_HEIGHT,
    overflowY: 'auto' as const,
    width: 0,
    minWidth: '100%',
    margin: 0,
    whiteSpace: 'pre-wrap' as const,
    overflowWrap: 'anywhere' as const,
    fontFamily: 'monospace',
    fontSize: 12,
};

// entryToText renders the whole entry as one JSON object for the copy action.
function entryToText(entry: LogEntry): string {
    const meta = [
        `  "timestamp": ${JSON.stringify(entry.timestamp)}`,
        `  "level": ${JSON.stringify(entry.level)}`,
    ];
    if (entry.caller) {
        meta.push(`  "caller": ${JSON.stringify(entry.caller)}`);
    }
    meta.push(`  "message": ${JSON.stringify(entry.message)}`);
    if (entry.stack) {
        meta.push(`  "stack": ${JSON.stringify(entry.stack)}`);
    }

    // Splice the server-formatted fields object's inner entries in beside the metadata.
    const fieldsInner = entry.fields && entry.fields.startsWith('{\n') && entry.fields.endsWith('\n}')
        ? entry.fields.slice(2, -2)
        : '';

    return fieldsInner
        ? `{\n${meta.join(',\n')},\n${fieldsInner}\n}`
        : `{\n${meta.join(',\n')}\n}`;
}

function LogEntryDetail({ entry }: { entry: LogEntry }) {
    return (
        <Box sx={{ py: 1.5 }}>
            {/* Panel-level action: copies the whole entry. stopPropagation so it doesn't collapse the row. */}
            <Box sx={{ display: 'flex', justifyContent: 'flex-end' }} onClick={(e) => e.stopPropagation()}>
                <CopyButton data={entryToText(entry)} toolTip="Copy entire log" />
            </Box>
            {entry.fields && (
                <>
                    <Typography variant="caption" color="textSecondary" sx={{ display: 'block', mb: 0.5 }}>
                        Fields
                    </Typography>
                    <Box sx={detailBlock}>
                        <SyntaxHighlighter
                            language="json"
                            style={prismTheme}
                            wrapLongLines
                            customStyle={{ margin: 0, borderRadius: 4, fontSize: 12 }}
                        >
                            {entry.fields}
                        </SyntaxHighlighter>
                    </Box>
                </>
            )}
            {entry.stack && (
                <Box sx={{ mt: 1.5 }}>
                    <Typography variant="caption" color="textSecondary" sx={{ display: 'block', mb: 0.5 }}>
                        Stack trace
                    </Typography>
                    <Box
                        component="pre"
                        sx={{ ...detailBlock, p: 1.5, borderRadius: 1, border: 1, borderColor: 'divider', bgcolor: 'action.hover' }}
                    >
                        {entry.stack}
                    </Box>
                </Box>
            )}
        </Box>
    );
}

const LogEntryRow = memo(function LogEntryRow({ entry, isExpanded, onToggle, mobile, isNew }: LogEntryRowProps) {
    const theme = useTheme();
    // The full message shows in the row; the dropdown only carries fields/stack.
    const hasExpandContent = entry.fields !== null || entry.stack !== null;
    const toggle = hasExpandContent ? () => onToggle(entry.id) : undefined;
    const expandIcon = hasExpandContent && (
        isExpanded ? <KeyboardArrowUpIcon fontSize="small" /> : <KeyboardArrowDownIcon fontSize="small" />
    );

    // Brief tint when a row arrives via live tail.
    const highlightSx = isNew
        ? {
            animation: 'logArrived 2s ease-out',
            '@keyframes logArrived': {
                from: { backgroundColor: alpha(theme.palette.info.main, 0.35) },
                to: { backgroundColor: 'transparent' },
            },
        }
        : undefined;

    if (mobile) {
        return (
            <Box
                component="li"
                onClick={toggle}
                sx={{
                    listStyle: 'none',
                    px: 2,
                    py: 1.5,
                    borderBottom: 1,
                    borderColor: 'divider',
                    cursor: toggle ? 'pointer' : undefined,
                    ...(highlightSx ?? {}),
                }}
            >
                <Box display="flex" alignItems="center" gap={1}>
                    <Chip label={entry.level} color={levelColor(entry.level)} size="small" />
                    <Timestamp timestamp={entry.timestamp} format="relative" variant="caption" color="textSecondary" />
                    <Box sx={{ flexGrow: 1 }} />
                    <Box sx={{ color: 'text.secondary', display: 'flex' }}>{expandIcon}</Box>
                </Box>
                {entry.caller && (
                    <Typography variant="caption" color="textSecondary" sx={{ display: 'block', mt: 0.5, fontFamily: 'monospace', wordBreak: 'break-all' }}>
                        {entry.caller}
                    </Typography>
                )}
                <Typography variant="body2" sx={{ mt: 0.5, wordBreak: 'break-word' }}>{entry.message}</Typography>
                <Collapse in={isExpanded} unmountOnExit>
                    <LogEntryDetail entry={entry} />
                </Collapse>
            </Box>
        );
    }

    return (
        <Fragment>
            <TableRow hover onClick={toggle} sx={{ cursor: toggle ? 'pointer' : undefined, ...(highlightSx ?? {}) }}>
                <TableCell sx={{ color: 'text.secondary' }}>{expandIcon}</TableCell>
                <TableCell>
                    <Chip label={entry.level} color={levelColor(entry.level)} size="small" />
                </TableCell>
                <TableCell>
                    <Timestamp timestamp={entry.timestamp} format="relative" variant="caption" noWrap />
                </TableCell>
                <TableCell>
                    {entry.caller && (
                        <Typography
                            variant="caption"
                            color="textSecondary"
                            noWrap
                            title={entry.caller}
                            sx={{ display: 'block', maxWidth: 220, fontFamily: 'monospace' }}
                        >
                            {entry.caller}
                        </Typography>
                    )}
                </TableCell>
                <TableCell sx={{ width: '100%' }}>
                    <Typography variant="body2" sx={{ wordBreak: 'break-word' }}>{entry.message}</Typography>
                </TableCell>
            </TableRow>
            <TableRow>
                <TableCell colSpan={5} sx={{ py: 0, border: 0 }}>
                    <Collapse in={isExpanded} unmountOnExit>
                        <Box sx={{ px: 2, borderBottom: 1, borderColor: 'divider' }}>
                            <LogEntryDetail entry={entry} />
                        </Box>
                    </Collapse>
                </TableCell>
            </TableRow>
        </Fragment>
    );
});

function AdminAreaLogsContent() {
    const theme = useTheme();
    const mobile = useMediaQuery(theme.breakpoints.down('md'));
    const environment = useRelayEnvironment();
    const [searchParams, setSearchParams] = useSearchParams();

    // Seed filters from the URL so a filtered view is shareable and survives refresh.
    const initialLevels = (searchParams.get('levels')?.split(',') ?? [])
        .filter((l): l is AdminLogTailLevel => (LOG_LEVELS as string[]).includes(l));
    const initialSearch = searchParams.get('search') ?? '';

    const [levels, setLevels] = useState<AdminLogTailLevel[]>(initialLevels);
    const [liveEntries, setLiveEntries] = useState<LogEntry[]>([]);
    const pendingLiveEntries = useRef<LogEntry[]>([]);
    const [liveTail, setLiveTail] = useState(false);
    const [search, setSearch] = useState(initialSearch);
    const [appliedLevels, setAppliedLevels] = useState<AdminLogTailLevel[]>(initialLevels);
    const [appliedSearch, setAppliedSearch] = useState(initialSearch);
    const [subscriptionError, setSubscriptionError] = useState<string | null>(null);
    const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set());

    const queryData = useLazyLoadQuery<AdminAreaLogsQuery>(
        logsQuery,
        {
            limit: INITIAL_ITEM_COUNT,
            levels: appliedLevels.length ? appliedLevels : undefined,
            search: appliedSearch || undefined,
        },
        { fetchPolicy: 'store-and-network' }
    );

    const isDisabled = queryData.config.adminLogTailStorePluginType.toLowerCase() === 'noop';

    // Throttled refetch on filter change (2s, trailing), matching other list searches.
    const refetch = useMemo(
        () =>
            throttle(
                (newLevels: AdminLogTailLevel[], newSearch: string) => {
                    const normalizedSearch = newSearch.trim();
                    fetchQuery(environment, logsQuery, {
                        limit: INITIAL_ITEM_COUNT,
                        levels: newLevels.length ? newLevels : undefined,
                        search: normalizedSearch || undefined,
                    }).subscribe({
                        complete: () => {
                            setAppliedLevels(newLevels);
                            setAppliedSearch(normalizedSearch);
                            setLiveEntries([]);
                            setExpandedIds(new Set());
                        },
                        error: (err: Error) => {
                            setSubscriptionError(err?.message ?? 'Failed to load logs');
                        },
                    });
                },
                2000,
                { leading: false, trailing: true }
            ),
        [environment],
    );

    useEffect(() => () => { refetch.cancel(); }, [refetch]);

    const onLevelsChange = (event: SelectChangeEvent<AdminLogTailLevel[]>) => {
        const value = event.target.value;
        const newLevels = (typeof value === 'string' ? value.split(',') : value) as AdminLogTailLevel[];
        setLevels(newLevels);
        pendingLiveEntries.current = [];
        setLiveEntries([]);
        // Apply level immediately rather than waiting out the search throttle.
        refetch(newLevels, search);
        refetch.flush();
    };

    const onSearchChange = (event: ChangeEvent<HTMLInputElement>) => {
        const newSearch = event.target.value;
        setSearch(newSearch);
        refetch(levels, newSearch);
    };

    const onSearchKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
        if (event.key === 'Enter') {
            refetch.flush();
        }
    };

    const onLiveEntry = useCallback((entry: LogEntry) => {
        pendingLiveEntries.current.unshift(entry);
        if (pendingLiveEntries.current.length > MAX_PENDING_ENTRIES) {
            pendingLiveEntries.current.pop();
        }
    }, []);

    useEffect(() => {
        if (!liveTail) return;
        const id = setInterval(() => {
            if (pendingLiveEntries.current.length === 0) return;
            const batch = pendingLiveEntries.current;
            pendingLiveEntries.current = [];
            setLiveEntries(prev => [...batch, ...prev].slice(0, MAX_LIVE_ENTRIES));
        }, LIVE_FLUSH_INTERVAL_MS);
        return () => clearInterval(id);
    }, [liveTail]);

    const onSubscriptionError = useCallback((error: Error) => {
        setSubscriptionError(`Live tail error: ${error.message}`);
    }, []);

    const toggleExpanded = useCallback((id: string) => {
        setExpandedIds(prev => {
            const next = new Set(prev);
            if (next.has(id)) {
                next.delete(id);
            } else {
                next.add(id);
            }
            return next;
        });
    }, []);

    const staticEntries = useMemo(
        () => (queryData?.adminLogTail ?? []).map(n => ({
            id: n.id,
            timestamp: n.timestamp as string,
            level: n.level,
            message: n.message,
            caller: n.caller ?? null,
            stack: n.stack ?? null,
            fields: n.fields ?? null,
        })),
        [queryData?.adminLogTail]
    );

    // Live entries get the arrival highlight; also reused by combined to dedupe vs. static.
    const liveIds = useMemo(() => new Set(liveEntries.map(e => e.id)), [liveEntries]);

    // Guard live entries against the active filter (immediate levels, applied search) so the
    // previous subscription can't leak non-matching entries during a resubscribe.
    const combined = useMemo(() => {
        const liveMatching = liveEntries.filter(e => matchesFilter(e, levels, appliedSearch));
        return [...liveMatching, ...staticEntries.filter(e => !liveIds.has(e.id))];
    }, [liveEntries, staticEntries, liveIds, levels, appliedSearch]);

    // Reflect the applied filters in the URL (replace, so it doesn't spam history).
    useEffect(() => {
        const next = new URLSearchParams();
        if (appliedLevels.length) {
            next.set('levels', appliedLevels.join(','));
        }
        if (appliedSearch) {
            next.set('search', appliedSearch);
        }
        setSearchParams(next, { replace: true });
    }, [appliedLevels, appliedSearch, setSearchParams]);

    if (isDisabled) {
        return (
            <Box>
                <AdminAreaBreadcrumbs childRoutes={BREADCRUMB_ROUTES} />
                <Typography variant="h5" gutterBottom>API Logs</Typography>
                <Typography variant="body2" sx={{ marginBottom: 3 }}>{DESCRIPTION}</Typography>
                <Box display="flex" flexDirection="column" alignItems="center" padding={4}>
                    <Typography variant="h6">API log tailing is disabled</Typography>
                    <Typography color="textSecondary" align="center">
                        Set THARSIS_ADMIN_LOG_TAIL_STORE_PLUGIN_TYPE to memory or redis to enable it.
                    </Typography>
                </Box>
            </Box>
        );
    }

    return (
        <Box>
            {liveTail && <LiveTailSubscriber levels={levels} search={appliedSearch} onEntry={onLiveEntry} onError={onSubscriptionError} />}
            {subscriptionError && (
                <Alert severity="error" sx={{ mb: 2 }}>
                    {subscriptionError}
                </Alert>
            )}
            <AdminAreaBreadcrumbs childRoutes={BREADCRUMB_ROUTES} />
            <Typography variant="h5" gutterBottom>API Logs</Typography>
            <Typography variant="body2" sx={{ marginBottom: 2 }}>{DESCRIPTION}</Typography>

            <Paper sx={{
                borderBottomLeftRadius: 0,
                borderBottomRightRadius: 0,
                border: 1,
                borderColor: 'divider',
            }}>
                <Box
                    padding={2}
                    display="flex"
                    gap={2}
                    flexDirection={mobile ? 'column' : 'row'}
                    alignItems={mobile ? 'stretch' : 'center'}
                >
                    <FormControl size="small" sx={{ minWidth: 160 }}>
                        <InputLabel shrink>Levels</InputLabel>
                        <Select
                            multiple
                            displayEmpty
                            value={levels}
                            label="Levels"
                            onChange={onLevelsChange}
                            renderValue={(selected) => (selected.length ? selected.join(', ') : 'All')}
                        >
                            {LOG_LEVELS.map(l => (
                                <MenuItem key={l} value={l}>
                                    <Checkbox checked={levels.includes(l)} size="small" />
                                    <ListItemText primary={l} />
                                </MenuItem>
                            ))}
                        </Select>
                    </FormControl>

                    <Box sx={{ flexGrow: 1, minWidth: 0, display: 'flex', justifyContent: mobile ? 'flex-start' : 'center' }}>
                        <SearchInput
                            placeholder="Search message, caller, and fields"
                            value={search}
                            onChange={onSearchChange}
                            onKeyDown={onSearchKeyDown}
                            inputProps={{ maxLength: 256 }}
                            sx={{ width: '100%', maxWidth: mobile ? 'none' : 480 }}
                        />
                    </Box>

                    <Box display="flex" alignItems="center" gap={1} justifyContent={mobile ? 'space-between' : undefined}>
                        <Typography variant="body2" color="textSecondary">Live tail</Typography>
                        <Switch
                            checked={liveTail}
                            onChange={(e: ChangeEvent<HTMLInputElement>) => {
                                setLiveTail(e.target.checked);
                                // Freeze: keep streamed entries, drop only the un-flushed buffer.
                                if (!e.target.checked) {
                                    pendingLiveEntries.current = [];
                                }
                            }}
                            size="small"
                        />
                    </Box>
                </Box>
            </Paper>

            <Box sx={{
                borderBottom: 1,
                borderLeft: 1,
                borderRight: 1,
                borderColor: 'divider',
                borderBottomLeftRadius: 4,
                borderBottomRightRadius: 4,
            }}>
                {mobile ? (
                    <Box>
                        {combined.length === 0 ? (
                            <Typography color="textSecondary" align="center" sx={{ p: 2 }}>No log entries</Typography>
                        ) : (
                            <List disablePadding>
                                {combined.map(entry => (
                                    <LogEntryRow
                                        key={entry.id}
                                        mobile
                                        entry={entry}
                                        isExpanded={expandedIds.has(entry.id)}
                                        onToggle={toggleExpanded}
                                        isNew={liveIds.has(entry.id)}
                                    />
                                ))}
                            </List>
                        )}
                    </Box>
                ) : (
                    <TableContainer>
                        <Table size="small">
                            <TableHead>
                                <TableRow>
                                    <TableCell />
                                    <TableCell><Typography color="textSecondary">Level</Typography></TableCell>
                                    <TableCell><Typography color="textSecondary">Timestamp</Typography></TableCell>
                                    <TableCell><Typography color="textSecondary">Caller</Typography></TableCell>
                                    <TableCell><Typography color="textSecondary">Message</Typography></TableCell>
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {combined.length === 0 && (
                                    <TableRow>
                                        <TableCell colSpan={5} align="center">
                                            <Typography color="textSecondary" padding={2}>No log entries</Typography>
                                        </TableCell>
                                    </TableRow>
                                )}
                                {combined.map(entry => (
                                    <LogEntryRow
                                        key={entry.id}
                                        mobile={false}
                                        entry={entry}
                                        isExpanded={expandedIds.has(entry.id)}
                                        onToggle={toggleExpanded}
                                        isNew={liveIds.has(entry.id)}
                                    />
                                ))}
                            </TableBody>
                        </Table>
                    </TableContainer>
                )}
            </Box>
        </Box>
    );
}

function AdminAreaLogs() {
    return (
        <Suspense fallback={
            <Box>
                <AdminAreaBreadcrumbs childRoutes={BREADCRUMB_ROUTES} />
                <Typography variant="h5" sx={{ marginBottom: 2 }}>API Logs</Typography>
                <Box display="flex" justifyContent="center" paddingTop={4}>
                    <CircularProgress size={24} />
                </Box>
            </Box>
        }>
            <AdminAreaLogsContent />
        </Suspense>
    );
}

export default AdminAreaLogs;
