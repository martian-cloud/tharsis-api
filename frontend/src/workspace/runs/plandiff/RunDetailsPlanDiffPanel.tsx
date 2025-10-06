import ChevronRightIcon from '@mui/icons-material/ChevronRight';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { Alert, Chip, Collapse, IconButton, Paper, Typography, useTheme } from '@mui/material';
import Box from '@mui/material/Box';
import 'prism-themes/themes/prism-holi-theme.css';
import { useMemo } from 'react';
import { ChangeData, Diff, DiffType, expandCollapsedBlockBy, getChangeKey, Hunk, HunkData, parseDiff, textLinesToHunk, useTokenizeWorker } from 'react-diff-view';
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

// useExpandedHunks is a hook which will expand all the collapsed hunks in the diff
function useExpandedHunks(hunks: HunkData[], source: string): HunkData[] {
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

            return expandCollapsedBlockBy(processedHunks, source, () => true);
        },
        [hunks, source]
    );
    return renderingHunks;
}

function DiffView({ diffType, hunks, oldSrc, warnings }: { diffType: DiffType, hunks: HunkData[], oldSrc: string, warnings: readonly PlanChangeWarning[] }) {
    const processedHunks = useExpandedHunks(hunks, oldSrc);
    const widgets = useWidgets(processedHunks, warnings);

    const { tokens } = useTokenizeWorker(tokenizeWorker, {
        oldSource: oldSrc,
        language: 'hcl',
        hunks: processedHunks,
        enhancers: []
    });

    return (
        <Diff
            viewType="unified"
            diffType={diffType}
            hunks={processedHunks}
            gutterType="none"
            tokens={tokens}
            widgets={widgets}
        >
            {hunks => hunks.flatMap(hunk => <Hunk key={hunk.content} hunk={hunk} />)}
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
                <DiffActionChip action={action} importing={imported} />
            </Box>
            <Collapse in={!collapsed} timeout="auto" unmountOnExit>
                <Box sx={{ backgroundColor: 'rgb(29, 31, 33)', fontSize: 14, fontFamily: theme.typography.code.fontFamily }}>
                    <DiffView hunks={file ? file.hunks : []} diffType={file ? file.type : 'modify'} oldSrc={oldSrc} warnings={warnings} />
                </Box>
            </Collapse>
        </Paper>
    );
}

export default RunDetailsPlanDiffPanel;
