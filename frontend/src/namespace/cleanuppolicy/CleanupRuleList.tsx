import { useState } from 'react';
import AddIcon from '@mui/icons-material/Add';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import {
    Box, Button, Divider, Paper, Stack, Tooltip, Typography, useMediaQuery, useTheme,
} from '@mui/material';
import { ResponsiveTable } from '../../common/ResponsiveTable';
import CleanupRuleListItem from './CleanupRuleListItem';
import { CleanupRule, Condition } from './types';
import { moveRule } from './utils';

const COLUMNS = [
    { label: 'Order' },
    { label: 'Outcome' },
    { label: 'Applies To' },
    { label: '', align: 'right' as const },
];

// Kept as a single source of truth since both the hint's own card-mode check below and the
// ResponsiveTable prop need to switch at the exact same breakpoint.
const CARD_MODE_BREAKPOINT = 'sm' as const;

export interface CleanupRuleTemplate<R extends CleanupRule> {
    label: string;
    detail: string;
    rule: () => R;
}

interface Props<R extends CleanupRule> {
    rules: readonly R[];
    onChange?: (rules: readonly R[]) => void;
    resource: string;
    ariaLabel: string;
    renderDialog: (props: {
        rule: R;
        position: number;
        isNew?: boolean;
        open: boolean;
        onClose: () => void;
        onSave: (rule: R) => void;
    }) => React.ReactNode;
    getConditions: (rule: CleanupRule) => Condition[];
    adding: R | undefined;
    onAddingChange: (rule: R | undefined) => void;
    onRequestAdd: () => void;
}

function CleanupRuleList<R extends CleanupRule>({
    rules, onChange, resource, ariaLabel,
    renderDialog, getConditions,
    adding, onAddingChange, onRequestAdd,
}: Props<R>) {
    const [editing, setEditing] = useState<number>();
    const readOnly = !onChange;
    // ResponsiveTable only renders an actual table header (for the overlay below to sit inside)
    // above this breakpoint — below it, rules render as a stack of cards with no header row, so
    // the same absolute overlay would land on top of the first card instead.
    const theme = useTheme();
    const cardMode = useMediaQuery(theme.breakpoints.down(CARD_MODE_BREAKPOINT));

    // A new PROTECT rule appended at the end would violate the rule requiring every PROTECT rule to
    // precede every non-PROTECT rule, so it's inserted just before the first non-PROTECT rule instead.
    const addRule = (rule: R) => {
        if (!onChange) return;
        if (rule.strategy === 'PROTECT') {
            const firstNonProtect = rules.findIndex(r => r.strategy !== 'PROTECT');
            const index = firstNonProtect === -1 ? rules.length : firstNonProtect;
            onChange([...rules.slice(0, index), rule, ...rules.slice(index)]);
        } else {
            onChange([...rules, rule]);
        }
        onAddingChange(undefined);
    };

    return (
        <Box>
            {rules.length > 0 && (
                <Box sx={{ position: 'relative' }}>
                    {/* In table mode (desktop) the hint overlays the header row at a fixed right
                        position; in card mode (mobile) there's no header row to overlay, so it
                        renders in-flow above the cards instead, right-aligned to match. */}
                    {cardMode ? (
                        <Box sx={{ display: 'flex', justifyContent: 'flex-end', mb: 1 }}>
                            <Tooltip title="Rules are checked in order; the first match decides the outcome.">
                                <Stack direction="row" alignItems="center" spacing={0.5} sx={{ cursor: 'default' }}>
                                    <InfoOutlinedIcon sx={{ fontSize: 14, color: 'text.secondary' }} />
                                    <Typography variant="caption" color="textSecondary">First matching rule wins</Typography>
                                </Stack>
                            </Tooltip>
                        </Box>
                    ) : (
                        <Box sx={{ position: 'absolute', right: 16, top: 0, height: 53, display: 'flex', alignItems: 'center', zIndex: 1, pointerEvents: 'none' }}>
                            <Box sx={{ pointerEvents: 'auto' }}>
                                <Tooltip title="Rules are checked in order; the first match decides the outcome.">
                                    <Stack direction="row" alignItems="center" spacing={0.5} sx={{ cursor: 'default' }}>
                                        <InfoOutlinedIcon sx={{ fontSize: 14, color: 'text.secondary' }} />
                                        <Typography variant="caption" color="textSecondary">First matching rule wins</Typography>
                                    </Stack>
                                </Tooltip>
                            </Box>
                        </Box>
                    )}
                    <ResponsiveTable ariaLabel={ariaLabel} columns={COLUMNS} breakpoint={CARD_MODE_BREAKPOINT}>
                        {rules.map((rule, i) => (
                            <CleanupRuleListItem
                                key={i} rule={rule} resource={resource} index={i + 1}
                                readOnly={readOnly}
                                conditions={getConditions(rule)}
                                canMoveUp={i > 0 && !(rules[i - 1].strategy === 'PROTECT' && rule.strategy !== 'PROTECT')}
                                canMoveDown={i < rules.length - 1 && !(rule.strategy === 'PROTECT' && rules[i + 1].strategy !== 'PROTECT')}
                                onEdit={() => setEditing(i)}
                                onMove={onChange ? offset => onChange(moveRule(rules, i, offset)) : undefined}
                                onDelete={onChange ? () => onChange(rules.filter((_, j) => j !== i)) : undefined}
                            />
                        ))}
                    </ResponsiveTable>
                </Box>
            )}

            {rules.length === 0 && (
                <Paper sx={{ p: 2 }}>
                    <Typography>No cleanup rules are configured.</Typography>
                </Paper>
            )}

            {!readOnly && (
                <>
                    <Divider sx={{ mt: 2 }} />
                    <Button
                        fullWidth size="small" color="inherit" variant="outlined"
                        startIcon={<AddIcon />}
                        onClick={onRequestAdd}
                        sx={{ mt: 2, borderStyle: 'dashed', color: 'text.secondary', borderColor: 'divider' }}
                    >
                        Add rule
                    </Button>
                </>
            )}

            {adding && renderDialog({
                rule: adding, position: rules.length + 1, isNew: true, open: true,
                onClose: () => onAddingChange(undefined),
                onSave: addRule,
            })}
            {editing !== undefined && renderDialog({
                rule: rules[editing], position: editing + 1, open: true,
                onClose: () => setEditing(undefined),
                onSave: rule => {
                    if (!onChange) { setEditing(undefined); return; }
                    const mapped = rules.map((r, i) => i === editing ? rule : r);
                    // Reposition whenever the rule crosses the PROTECT/non-PROTECT boundary in
                    // either direction: new strategy is PROTECT (move before first non-PROTECT),
                    // or original strategy was PROTECT (move to after last remaining PROTECT rule).
                    if (rule.strategy === 'PROTECT' || rules[editing].strategy === 'PROTECT') {
                        const without = mapped.filter((_, i) => i !== editing);
                        const firstNonProtect = without.findIndex(r => r.strategy !== 'PROTECT');
                        const index = firstNonProtect === -1 ? without.length : firstNonProtect;
                        onChange([...without.slice(0, index), rule, ...without.slice(index)]);
                    } else {
                        onChange(mapped);
                    }
                    setEditing(undefined);
                },
            })}
        </Box>
    );
}

export default CleanupRuleList;
