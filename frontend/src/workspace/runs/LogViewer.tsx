import { Box, SxProps, Theme, useTheme } from '@mui/material';
import grey from '@mui/material/colors/grey';
import Anser from 'anser';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import Link from '../../routes/Link';

interface Props {
    logs: string
    hideLineNumbers?: boolean
    sx?: SxProps<Theme>
}

interface LogLineProps {
    log: string
    lineNumber: number
    selected?: boolean
    hideLineNumber?: boolean
}

function buildLogTextStyle(entry: Anser.AnserJsonEntry) {
    return {
        backgroundColor: entry.bg ? `rgb(${entry.bg})` : undefined,
        color: entry.fg ? `rgb(${entry.fg})` : undefined,
        fontWeight: entry.decoration === 'bold' ? 700 : undefined
    };
}

function LogLine({ log, lineNumber, selected, hideLineNumber }: LogLineProps) {
    const theme = useTheme();
    const ref = useRef<HTMLDivElement>();
    const [autoScroll, setAutoscroll] = useState(true);

    const parts = useMemo(() => Anser.ansiToJson(log).filter(part => part.content !== '').
        map((part, index) => (<Box
            key={index}
            component="span"
            sx={buildLogTextStyle(part)}
        >
            {part.content}
        </Box>)), [log]);

    useEffect(() => {
        if (selected && autoScroll && ref.current) {
            ref.current.scrollIntoView({
                block: 'center'
            });
        }
    }, [selected, autoScroll]);

    return <Box ref={ref} sx={{ padding: `1px 0px 1px ${hideLineNumber ? '0px' : '56px'}` }}>
        {!hideLineNumber && <Link
            id={`L${lineNumber}`}
            preventScrollReset
            sx={{
                color: selected ? theme.palette.primary.main : grey[500],
                marginLeft: '-48px',
                paddingLeft: 1,
                paddingRight: 2,
                minWidth: '48px',
                textAlign: 'right',
                display: 'inline-block',
            }}
            replace
            state={{
                preventScrollReset: true
            }}
            to={{
                search: `line=${lineNumber}`,
            }}
            onClick={() => setAutoscroll(false)}
        >
            {lineNumber}
        </Link>}
        {parts}
    </Box>;
}

const MemorizedLogLine = React.memo(LogLine);

function LogViewer({ logs, sx, hideLineNumbers }: Props) {
    const theme = useTheme();
    const [searchParams] = useSearchParams();

    const [selectedLine, setSelectedLine] = useState<number | undefined>();

    useEffect(() => {
        if (hideLineNumbers) {
            // Line selection is not supported when line numbers are hidden
            return;
        }
        const selectedLineParam = searchParams.get('line');
        if (selectedLineParam) {
            setSelectedLine(parseInt(selectedLineParam));
        }
    }, [searchParams]);

    const logLines = useMemo(() => {
        const lines = logs.split(/\r?\n/);
        if (lines.length === 1 && lines[0] === '') {
            return [];
        }

        // ANSI codes don't carry across newlines after splitting, so we track the active
        // styles and prepend them to subsequent lines until a reset code is encountered.
        // This ensures multi-line colored output (e.g., Terraform's green success messages)
        // maintains consistent styling across line breaks.
        const esc = '\u001b';
        const ansiCodeRegex = new RegExp(`${esc}\\[[0-9;]*m`, 'g');
        const isResetCode = (code: string) => code === `${esc}[0m` || code === `${esc}[0;m`;

        let activeCodes: string[] = [];
        return lines.map(line => {
            const result = activeCodes.join('') + line;
            const codes = line.match(ansiCodeRegex) || [];
            for (const code of codes) {
                if (isResetCode(code)) {
                    activeCodes = [];
                } else {
                    activeCodes.push(code);
                }
            }
            return result;
        });
    }, [logs]);

    return (
        <Box
            sx={{
                fontSize: '13px',
                fontFamily: 'ui-monospace,SFMono-Regular,SF Mono,Menlo,Consolas,Liberation Mono,monospace !important',
                color: theme.palette.text.primary,
                wordBreak: 'break-all',
                whiteSpace: 'pre-wrap',
                mt: 0,
                mb: 0,
                height: '100%',
                overflowY: 'auto',
                ...sx
            }}
            component="pre"
        >
            {logLines.map((l, index) => {
                const lineNumber = index + 1;
                return <MemorizedLogLine key={`L${lineNumber}`} log={l} lineNumber={lineNumber} selected={lineNumber === selectedLine} hideLineNumber={hideLineNumbers} />;
            })}
        </Box>
    );
}

export default LogViewer;
