import ChevronRightIcon from '@mui/icons-material/ChevronRight';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import { Alert, Chip, Collapse, IconButton, Menu, MenuItem, Paper, Typography, useTheme } from '@mui/material';
import Box from '@mui/material/Box';
import 'prism-themes/themes/prism-holi-theme.css';
import { useMemo, useState } from 'react';
import { ChangeData, Decoration, Diff, DiffType, expandFromRawCode, getChangeKey, Hunk, HunkData, parseDiff, textLinesToHunk, useTokenizeWorker } from 'react-diff-view';
import 'react-diff-view/style/index.css';
import * as refractor from 'refractor';
import hcl from 'refractor/lang/hcl';
import { PlanChangeAction, PlanChangeWarningType } from './__generated__/RunDetailsPlanDiffViewerFragment_run.graphql';
import DiffActionChip from './DiffActionChip';
import './diffview.css';
import colors from './RunDetailsPlanDiffColors';

// Register hcl language for syntax highlighting
refractor.register(hcl);

// Start tokenize working for syntax highlighting
const tokenizeWorker = new Worker(new URL('./Tokenize.ts', import.meta.url), { type: "module" });

type PlanChangeWarning = { readonly changeType: PlanChangeWarningType; readonly line: number; readonly message: string; };

// useWidgets is a hook which will return widgets for each warning in the diff
const useWidgets = (hunks: HunkData[], warnings: readonly PlanChangeWarning[]) => {
    return useMemo(() => {
        // Convert warnings to a map for faster lookup
        const warningsMap = warnings.reduce((result, warning) => {
            const key = `${warning.changeType}-${warning.line}`;
            // check if result already contains a warning for this line
            if (result[key]) {
                result[key] = `${result[key]}. ${warning.message}`;
                return result;
            }
            return { ...result, [key]: warning.message };
        }, {} as { [key: string]: string }) as { [key: string]: string };

        const changes = hunks.reduce((result: any, { changes }) => [...result, ...changes], []);
        return changes.reduce(
            (widgets: any, change: ChangeData) => {
                let warningMessage = null;
                // Check if there is a warning for this change
                switch (change.type) {
                    case 'insert':
                        // Check for warning in after file that matches this line number
                        warningMessage = warningsMap[`after-${change.lineNumber}`];
                        break;
                    case 'delete':
                        // Check for warning in before file that matches this line number
                        warningMessage = warningsMap[`before-${change.lineNumber}`];
                        break;
                    case 'normal':
                        // Check for warning in before or after file that matches this line number
                        warningMessage = warningsMap[`after-${change.newLineNumber}`];
                        break;
                }

                if (!warningMessage) {
                    return widgets;
                }

                const changeKey = getChangeKey(change);

                return {
                    ...widgets,
                    [changeKey]: <Alert severity="warning" key={changeKey} sx={{ p: `0 0 0 8px`, fontSize: 13 }}>{warningMessage}</Alert>
                };
            },
            {}
        );
    }, [hunks, warnings]);
};

// CollapsedRange is a run of unchanged source lines hidden between hunks, identified by the old line it starts at
type CollapsedRange = { oldStart: number, newStart: number, lines: number };

// getCollapsedRanges returns the collapsed range before each hunk, plus a final entry for the range after the
// last hunk, so the result has hunks.length + 1 entries; null means no lines are hidden there. It is the single
// place collapsed boundaries are computed, so the ranges that get expanded and the ranges that get rendered
// (and that a click expands) always agree.
function getCollapsedRanges(hunks: HunkData[], sourceLineCount: number): (CollapsedRange | null)[] {
    const ranges = hunks.map((hunk, index) => {
        const previousHunk = index > 0 ? hunks[index - 1] : null;
        const oldStart = previousHunk ? previousHunk.oldStart + previousHunk.oldLines : 1;
        const newStart = previousHunk ? previousHunk.newStart + previousHunk.newLines : 1;
        return { oldStart, newStart, lines: hunk.oldStart - oldStart };
    });

    const lastHunk = hunks[hunks.length - 1];
    if (lastHunk) {
        const oldStart = lastHunk.oldStart + lastHunk.oldLines;
        const newStart = lastHunk.newStart + lastHunk.newLines;
        ranges.push({ oldStart, newStart, lines: sourceLineCount - oldStart + 1 });
    }

    return ranges.map(range => range.lines > 0 ? range : null);
}

// warningInRange reports whether any warning falls on a line within the given collapsed range
function warningInRange(warnings: readonly PlanChangeWarning[], { lines, oldStart, newStart }: CollapsedRange): boolean {
    return warnings.some(({ changeType, line }) => changeType === 'before'
        ? line >= oldStart && line < oldStart + lines
        : line >= newStart && line < newStart + lines);
}

// useExpandedHunks is a hook which expands the collapsed blocks between hunks. When showFullContent is
// false, only the blocks the user has expanded, or that contain a warning, are expanded so the diff shows
// just the changes and their surrounding context lines.
function useExpandedHunks(hunks: HunkData[], source: string, warnings: readonly PlanChangeWarning[], showFullContent: boolean, expandedBlocks: Set<number>): HunkData[] {
    const renderingHunks = useMemo(
        () => {
            if (!source) {
                return hunks;
            }

            let processedHunks = hunks;

            if (hunks.length === 0) {
                const hunk = textLinesToHunk(source.split('\n'), 1, 1);
                processedHunks = hunk !== null ? [hunk] : [];
            }

            if (processedHunks.length === 0) {
                return processedHunks;
            }

            const sourceLines = source.split('\n');
            return getCollapsedRanges(processedHunks, sourceLines.length)
                .filter((range): range is CollapsedRange => range !== null)
                .filter(range => showFullContent || expandedBlocks.has(range.oldStart) || warningInRange(warnings, range))
                .reduce((result, range) => expandFromRawCode(result, sourceLines, range.oldStart, range.oldStart + range.lines), processedHunks);
        },
        [hunks, source, warnings, showFullContent, expandedBlocks]
    );
    return renderingHunks;
}

function CollapsedBlock({ lines, onExpand }: { lines: number, onExpand: () => void }) {
    return (
        <Decoration>
            <Box component="button" type="button" className="diff-collapsed" onClick={onExpand}>
                ⋯ {lines} unchanged {lines === 1 ? 'line' : 'lines'}
            </Box>
        </Decoration>
    );
}

function DiffView({ diffType, hunks, oldSrc, warnings, showFullContent }: { diffType: DiffType, hunks: HunkData[], oldSrc: string, warnings: readonly PlanChangeWarning[], showFullContent: boolean }) {
    const [expandedBlocks, setExpandedBlocks] = useState<Set<number>>(new Set());
    const processedHunks = useExpandedHunks(hunks, oldSrc, warnings, showFullContent, expandedBlocks);
    const widgets = useWidgets(processedHunks, warnings);

    const workerOptions = useMemo(() => {
        return {
            oldSource: oldSrc,
            language: 'hcl',
            hunks: processedHunks,
            enhancers: []
        };
    }, [oldSrc, processedHunks]);

    const { tokens } = useTokenizeWorker(tokenizeWorker, workerOptions);

    const sourceLineCount = useMemo(() => oldSrc ? oldSrc.split('\n').length : 0, [oldSrc]);

    const expandBlock = (oldStart: number) => setExpandedBlocks(prev => new Set(prev).add(oldStart));

    const renderHunks = (hunks: HunkData[]) => {
        // Collapsed blocks only exist relative to the original source
        if (!oldSrc) {
            return hunks.map(hunk => <Hunk key={hunk.content} hunk={hunk} />);
        }

        const ranges = getCollapsedRanges(hunks, sourceLineCount);
        const renderRange = (range: CollapsedRange | null) => range
            ? [<CollapsedBlock key={`collapsed-${range.oldStart}`} lines={range.lines} onExpand={() => expandBlock(range.oldStart)} />]
            : [];

        return [
            ...hunks.flatMap((hunk, index) => [...renderRange(ranges[index]), <Hunk key={hunk.content} hunk={hunk} />]),
            ...renderRange(ranges[hunks.length])
        ];
    };

    return (
        <Diff
            viewType="unified"
            diffType={diffType}
            hunks={processedHunks}
            gutterType="none"
            tokens={tokens}
            widgets={widgets}
        >
            {renderHunks}
        </Diff>
    );
}

export type Props = {
    title: string,
    action: PlanChangeAction,
    drift: boolean,
    imported: boolean,
    diff: string,
    oldSrc: string,
    warnings: readonly PlanChangeWarning[]
    collapsed: boolean
    onCollapseChange: (collapsed: boolean) => void
};

function RunDetailsPlanDiffPanel({ title, action, drift, imported, diff, oldSrc, warnings, collapsed, onCollapseChange }: Props) {
    const theme = useTheme();
    const [showFullContent, setShowFullContent] = useState<boolean>(false);
    const [menuAnchorEl, setMenuAnchorEl] = useState<HTMLElement | null>(null);

    const toggleFullContent = () => {
        setMenuAnchorEl(null);
        setShowFullContent(!showFullContent);
        // Showing the full contents of a collapsed panel opens it, so the change is visible.
        if (collapsed) {
            onCollapseChange(false);
        }
    };
    const file = useMemo(
        () => {
            const [file] = diff ? parseDiff(diff) : [];
            return file;
        },
        [diff]
    );

    return (
        <Paper key={title} sx={{ mb: 2 }} variant="outlined">
            <Box p={1} display="flex" justifyContent="space-between" alignItems="center">
                <Box display="flex" alignItems="center">
                    <IconButton size="small" onClick={() => onCollapseChange(!collapsed)}>
                        {!collapsed && <ExpandMoreIcon />}
                        {collapsed && <ChevronRightIcon />}
                    </IconButton>
                    <Typography variant="code" fontWeight={600}>{title}</Typography>
                    {drift && <Chip size="xs" label="drift" sx={{ color: colors.drift, ml: 1 }} />}
                </Box>
                <Box display="flex" alignItems="center">
                    <DiffActionChip action={action} importing={imported} />
                    {/* Unchanged lines can only be hidden relative to the original source. */}
                    {oldSrc && <>
                        <IconButton size="small" sx={{ ml: 0.5 }} aria-label="more options" onClick={e => setMenuAnchorEl(e.currentTarget)}>
                            <MoreVertIcon fontSize="small" />
                        </IconButton>
                        <Menu
                            anchorEl={menuAnchorEl}
                            open={!!menuAnchorEl}
                            onClose={() => setMenuAnchorEl(null)}
                            anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
                            transformOrigin={{ vertical: 'top', horizontal: 'right' }}
                        >
                            <MenuItem onClick={toggleFullContent}>
                                {showFullContent ? 'Show changes only' : 'Show full contents'}
                            </MenuItem>
                        </Menu>
                    </>}
                </Box>
            </Box>
            <Collapse in={!collapsed} timeout="auto" unmountOnExit>
                <Box sx={{ backgroundColor: 'rgb(29, 31, 33)', fontSize: 14, fontFamily: theme.typography.code.fontFamily }}>
                    <DiffView hunks={file ? file.hunks : []} diffType={file ? file.type : 'modify'} oldSrc={oldSrc} warnings={warnings} showFullContent={showFullContent} />
                </Box>
            </Collapse>
        </Paper>
    );
}

export default RunDetailsPlanDiffPanel;
