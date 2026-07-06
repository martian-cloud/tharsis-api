import { alpha, Box, SxProps, Theme, useTheme } from '@mui/material';
import { useVirtualizer, useWindowVirtualizer, Virtualizer } from '@tanstack/react-virtual';
import Anser from 'anser';
import React, { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import LoadingDots from '../../common/LoadingDots';

interface Props {
    logs: string
    hideLineNumbers?: boolean
    loading?: boolean
    followOutput?: boolean
    scrollMode?: 'window' | 'container'
    sx?: SxProps<Theme>
    disableDeepLink?: boolean
}

const LINE_HEIGHT_ESTIMATE = 22;
const OVERSCAN = 20;

const esc = '\u001b';
const ansiCodeRegex = new RegExp(`${esc}\\[[0-9;]*m`, 'g');
const isResetCode = (code: string) => code === `${esc}[0m` || code === `${esc}[0;m`;

function buildLogTextStyle(entry: Anser.AnserJsonEntry): React.CSSProperties {
    return {
        backgroundColor: entry.bg ? `rgb(${entry.bg})` : undefined,
        color: entry.fg ? `rgb(${entry.fg})` : undefined,
        fontWeight: entry.decoration === 'bold' ? 700 : undefined
    };
}

interface LogAccumulator {
    consumed: string    // the prefix of `logs` already split into committed lines + pending
    lines: string[]     // committed lines, each prefixed with the ANSI codes carried into it
    activeCodes: string[] // SGR codes active at the end of the committed lines
    pending: string     // trailing text after the last newline (incomplete line)
}

// Incrementally parses only the appended delta, carrying active ANSI codes across newlines,
// so the whole buffer is never re-scanned. The accumulator lives in a ref and is mutated in
// place; the `logs !== acc.consumed` guard makes a recompute with the same value a no-op,
// which keeps it safe under StrictMode's double-invoked useMemo. The returned `getLine` reads
// the live accumulator, so the {count, getLine} pair is only valid for the current `logs` —
// `count` changes whenever content does, which is what consumers key their effects off.
function useLogLines(logs: string) {
    const accRef = useRef<LogAccumulator>({ consumed: '', lines: [], activeCodes: [], pending: '' });

    return useMemo(() => {
        const acc = accRef.current;

        if (logs !== acc.consumed) {
            let delta: string;
            if (logs.startsWith(acc.consumed)) {
                delta = logs.slice(acc.consumed.length);
            } else {
                // Buffer was replaced or cleared, reprocess from scratch
                acc.lines = [];
                acc.activeCodes = [];
                acc.pending = '';
                delta = logs;
            }

            const parts = (acc.pending + delta).split(/\r?\n/);
            acc.pending = parts.pop() ?? '';
            for (const line of parts) {
                acc.lines.push(acc.activeCodes.join('') + line);
                const codes = line.match(ansiCodeRegex) || [];
                for (const code of codes) {
                    if (isResetCode(code)) {
                        acc.activeCodes = [];
                    } else {
                        acc.activeCodes.push(code);
                    }
                }
            }
            acc.consumed = logs;
        }

        const committed = acc.lines;
        const tail = acc.pending !== '' ? acc.activeCodes.join('') + acc.pending : null;
        const count = committed.length + (tail ? 1 : 0);
        const getLine = (index: number) => (index < committed.length ? committed[index] : (tail as string));
        return { count, getLine };
    }, [logs]);
}

interface LogRowProps {
    log: string
    lineNumber: number
    selected: boolean
    hideLineNumber?: boolean
    onSelect: (lineNumber: number) => void
}

function LogRow({ log, lineNumber, selected, hideLineNumber, onSelect }: LogRowProps) {
    const theme = useTheme();

    const parts = useMemo(() => Anser.ansiToJson(log).filter(part => part.content !== '').
        map((part, index) => (<span key={index} style={buildLogTextStyle(part)}>{part.content}</span>)), [log]);

    // Layout styles stay inline (synchronous) so the virtualizer measures the real wrapped
    // height; emotion/sx injects asynchronously and would cache a single-line height.
    return <div style={{
        padding: `1px 0px 1px ${hideLineNumber ? '0px' : '56px'}`,
        backgroundColor: selected ? alpha(theme.palette.primary.main, 0.15) : undefined,
        borderLeft: selected ? `3px solid ${theme.palette.primary.main}` : '3px solid transparent'
    }}>
        {!hideLineNumber && <a
            id={`L${lineNumber}`}
            href={`?line=${lineNumber}`}
            onClick={(event: React.MouseEvent) => {
                event.preventDefault();
                onSelect(lineNumber);
            }}
            style={{
                color: selected ? theme.palette.primary.main : theme.palette.grey[500],
                textDecoration: 'none',
                marginLeft: '-48px',
                paddingLeft: '8px',
                paddingRight: '16px',
                minWidth: '48px',
                textAlign: 'right',
                display: 'inline-block',
                cursor: 'pointer',
                userSelect: 'none',
            }}
        >
            {lineNumber}
        </a>}
        {parts}
    </div>;
}

const MemorizedLogRow = React.memo(LogRow);

// Inline so measurement is correct, see LogRow.
const rowWrapperStyle: React.CSSProperties = {
    width: '100%',
    fontSize: '13px',
    fontFamily: 'ui-monospace,SFMono-Regular,SF Mono,Menlo,Consolas,Liberation Mono,monospace',
    whiteSpace: 'pre-wrap',
    wordBreak: 'break-all'
};

interface LogListProps {
    virtualizer: Virtualizer<any, Element>
    scrollMargin: number
    count: number
    getLine: (index: number) => string
    selectedLine?: number
    onSelect: (lineNumber: number) => void
    lastScrolledLineRef: React.RefObject<number | undefined>
    hideLineNumbers?: boolean
    loading?: boolean
    followOutput?: boolean
}

// Rows render in normal flow inside one translated container so they can't overlap while
// scrolling fast (an unmeasured row just pushes the next one down).
function LogList({ virtualizer, scrollMargin, count, getLine, selectedLine, onSelect, lastScrolledLineRef, hideLineNumbers, loading, followOutput }: LogListProps) {
    useEffect(() => {
        if (followOutput && count > 0) {
            virtualizer.scrollToIndex(count - 1, { align: 'end' });
        }
    }, [virtualizer, count, followOutput]);

    // Scroll a deep-linked (?line=N) row into view once it loads; a clicked line is skipped.
    useEffect(() => {
        if (selectedLine === undefined || selectedLine > count || selectedLine === lastScrolledLineRef.current) {
            return;
        }
        virtualizer.scrollToIndex(selectedLine - 1, { align: 'center' });
        lastScrolledLineRef.current = selectedLine;
    }, [virtualizer, selectedLine, count, lastScrolledLineRef]);

    const items = virtualizer.getVirtualItems();
    const offset = (items[0]?.start ?? 0) - scrollMargin;

    return <>
        <Box sx={{ height: virtualizer.getTotalSize(), width: '100%', position: 'relative' }}>
            <div style={{ position: 'absolute', top: 0, left: 0, width: '100%', transform: `translateY(${offset}px)` }}>
                {items.map(item => {
                    const lineNumber = item.index + 1;
                    return <div key={item.key} data-index={item.index} ref={virtualizer.measureElement} style={rowWrapperStyle}>
                        <MemorizedLogRow
                            log={getLine(item.index)}
                            lineNumber={lineNumber}
                            selected={lineNumber === selectedLine}
                            hideLineNumber={hideLineNumbers}
                            onSelect={onSelect}
                        />
                    </div>;
                })}
            </div>
        </Box>
        {loading && <Box sx={{ marginLeft: '32px' }}>
            <LoadingDots />
        </Box>}
    </>;
}

interface ViewerProps {
    count: number
    getLine: (index: number) => string
    selectedLine?: number
    onSelect: (lineNumber: number) => void
    lastScrolledLineRef: React.RefObject<number | undefined>
    hideLineNumbers?: boolean
    loading?: boolean
    followOutput?: boolean
    sx?: SxProps<Theme>
}

const rootStyle = {
    fontSize: '13px',
    fontFamily: 'ui-monospace,SFMono-Regular,SF Mono,Menlo,Consolas,Liberation Mono,monospace !important',
    wordBreak: 'break-all',
    whiteSpace: 'pre-wrap',
    mt: 0,
    mb: 0
} as const;

function WindowLogViewer({ sx, ...rest }: ViewerProps) {
    const listRef = useRef<HTMLDivElement>(null);
    const [scrollMargin, setScrollMargin] = useState(0);

    // Track the list's document offset (used to position rows), refreshing if the chrome
    // above it resizes. Uses the viewport rect plus scroll position rather than offsetTop,
    // which is only relative to the offsetParent and would misplace rows when the list sits
    // inside a positioned ancestor.
    useLayoutEffect(() => {
        const update = () => setScrollMargin(prev => {
            const el = listRef.current;
            const next = el ? el.getBoundingClientRect().top + window.scrollY : 0;
            return prev === next ? prev : next;
        });
        update();
        const target = listRef.current?.offsetParent ?? document.body;
        const observer = new ResizeObserver(update);
        observer.observe(target);
        return () => observer.disconnect();
    }, []);

    const virtualizer = useWindowVirtualizer({
        count: rest.count,
        estimateSize: () => LINE_HEIGHT_ESTIMATE,
        overscan: OVERSCAN,
        scrollMargin
    });

    return <Box ref={listRef} sx={{ ...rootStyle, ...sx }}>
        <LogList virtualizer={virtualizer} scrollMargin={scrollMargin} {...rest} />
    </Box>;
}

function ContainerLogViewer({ sx, ...rest }: ViewerProps) {
    const scrollRef = useRef<HTMLDivElement>(null);

    const virtualizer = useVirtualizer({
        count: rest.count,
        getScrollElement: () => scrollRef.current,
        estimateSize: () => LINE_HEIGHT_ESTIMATE,
        overscan: OVERSCAN
    });

    return <Box ref={scrollRef} sx={{ ...rootStyle, height: '100%', overflowY: 'auto', ...sx }}>
        <LogList virtualizer={virtualizer} scrollMargin={0} {...rest} />
    </Box>;
}

function LogViewer({ logs, sx, hideLineNumbers, loading, followOutput, scrollMode, disableDeepLink }: Props) {
    const [searchParams, setSearchParams] = useSearchParams();
    const lastScrolledLineRef = useRef<number | undefined>(undefined);

    const selectedLine = useMemo(() => {
        if (hideLineNumbers || disableDeepLink) {
            return undefined;
        }
        const parsed = parseInt(searchParams.get('line') ?? '', 10);
        return Number.isInteger(parsed) && parsed >= 1 ? parsed : undefined;
    }, [searchParams, hideLineNumbers, disableDeepLink]);

    const onSelect = useCallback((lineNumber: number) => {
        if (disableDeepLink) {
            return;
        }
        // Record as already-scrolled so clicking a visible line doesn't yank the viewport
        lastScrolledLineRef.current = lineNumber;
        setSearchParams(prev => {
            const next = new URLSearchParams(prev);
            next.set('line', String(lineNumber));
            return next;
        }, { replace: true, preventScrollReset: true, state: { preventScrollReset: true } });
    }, [setSearchParams, disableDeepLink]);

    const { count, getLine } = useLogLines(logs);

    const viewerProps: ViewerProps = { count, getLine, selectedLine, onSelect, lastScrolledLineRef, hideLineNumbers, loading, followOutput, sx };

    return scrollMode === 'container' ? <ContainerLogViewer {...viewerProps} /> : <WindowLogViewer {...viewerProps} />;
}

export default LogViewer;
