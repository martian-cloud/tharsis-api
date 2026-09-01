import React, { useState } from 'react';
import CheckIcon from '@mui/icons-material/Check';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import {
    Box, Button, Dialog, DialogActions, DialogContent, DialogTitle,
    Stack, TextField, Typography, useMediaQuery, useTheme,
} from '@mui/material';
import { alpha } from '@mui/material/styles';
import CleanupRuleStrategyPicker from './CleanupRuleStrategyPicker';
import NumberBadge from './NumberBadge';
import { CleanupRule, CleanupStrategy } from './types';
import { CleanupRuleTemplate } from './CleanupRuleList';

// Number of templates shown before "See all N" expands the rest.
const TEMPLATE_PREVIEW_COUNT = 4;

interface Props<R extends CleanupRule> {
    rule: R;
    position: number;
    isNew?: boolean;
    open: boolean;
    resource: string;
    safetyNote?: React.ReactNode;
    onClose: () => void;
    onSave: (rule: R) => void;
    children: (rule: R, onChange: (rule: R) => void) => React.ReactNode;
    templates?: CleanupRuleTemplate<R>[];
    getSummary?: (rule: R) => React.ReactNode;
    availableStrategies?: CleanupStrategy[];
}

function CleanupRuleDialog<R extends CleanupRule>({
    rule: initialRule, position, isNew, open, resource, safetyNote, onClose, onSave,
    children, templates, getSummary, availableStrategies,
}: Props<R>) {
    const [draft, setDraft] = useState<R>(initialRule);
    const [showAllTemplates, setShowAllTemplates] = useState(false);
    const theme = useTheme();
    const fullScreen = useMediaQuery(theme.breakpoints.down('sm'));
    const visibleTemplates = !showAllTemplates && templates
        ? templates.slice(0, TEMPLATE_PREVIEW_COUNT)
        : templates ?? [];

    const applyTemplate = (t: CleanupRuleTemplate<R>) => setDraft(t.rule() as R);

    // Step numbers shift by one for a new rule: it has the template step, an existing rule does not.
    const stepOffset = isNew ? 1 : 0;

    return (
        // No onClose, since a click outside or Escape must not discard the draft — Cancel is the way out.
        <Dialog open={open} maxWidth="md" fullWidth fullScreen={fullScreen}>
            <DialogTitle>
                {isNew ? `Add ${resource} rule` : `${resource[0].toUpperCase()}${resource.slice(1)} rule ${position}`}
            </DialogTitle>
            {isNew && (
                <Typography variant="body2" color="textSecondary" sx={{ px: 3, pb: 2, mt: -2 }}>
                    The first rule that matches a {resource} decides what happens to it.
                </Typography>
            )}
            <DialogContent dividers>
                {/* Step 1 — templates (new rules only) */}
                {isNew && templates && templates.length > 0 && (
                    <Box sx={{ mb: 4 }}>
                        <Box sx={{ display: 'flex', alignItems: 'baseline', justifyContent: 'space-between', mb: 1 }}>
                            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                                <NumberBadge value={1} />
                                <Typography variant="subtitle2">
                                    Start from a template{' '}
                                    <Typography component="span" variant="caption" color="textSecondary">optional</Typography>
                                </Typography>
                            </Box>
                            {templates.length > TEMPLATE_PREVIEW_COUNT && (
                                <Button size="small" color="primary" onClick={() => setShowAllTemplates(v => !v)}>
                                    {showAllTemplates ? 'Show fewer' : `See all ${templates.length}`}
                                </Button>
                            )}
                        </Box>
                        <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: 1 }}>
                            {visibleTemplates.map(t => {
                                const selected = draft.description === t.rule().description &&
                                    draft.strategy === t.rule().strategy;
                                return (
                                    <Box
                                        key={t.label}
                                        onClick={() => applyTemplate(t)}
                                        sx={{
                                            p: 1.5, borderRadius: 1, cursor: 'pointer', position: 'relative',
                                            border: '1px solid',
                                            borderColor: selected ? 'primary.main' : 'divider',
                                            bgcolor: selected ? alpha(theme.palette.primary.main, 0.16) : 'transparent',
                                            '&:hover': { borderColor: selected ? 'primary.main' : 'text.disabled' },
                                        }}
                                    >
                                        {selected && (
                                            <CheckIcon
                                                fontSize="small"
                                                sx={{ position: 'absolute', top: 6, right: 6, color: 'primary.main' }}
                                            />
                                        )}
                                        <Typography variant="body2" fontWeight={500} pr={selected ? 2.5 : 0}>
                                            {t.label}
                                        </Typography>
                                        <Typography variant="caption" color="textSecondary">{t.detail}</Typography>
                                    </Box>
                                );
                            })}
                        </Box>
                    </Box>
                )}
                {/* Step 2 — strategy */}
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                    <NumberBadge value={1 + stepOffset} />
                    <Typography variant="subtitle2">What happens to a match</Typography>
                </Box>
                <CleanupRuleStrategyPicker rule={draft} onChange={r => setDraft(r as R)} availableStrategies={availableStrategies} />
                {/* Step 3 — filters */}
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mt: 4, mb: 1 }}>
                    <NumberBadge value={2 + stepOffset} />
                    <Typography variant="subtitle2">Which {resource}s it applies to</Typography>
                </Box>
                {children(draft, setDraft)}
                <TextField
                    sx={{ mt: 4 }} size="small" fullWidth label="Description"
                    placeholder={draft.description || `${draft.strategy ? 'describe this rule' : ''}`}
                    value={draft.description}
                    onChange={e => setDraft({ ...draft, description: e.target.value })}
                />
                {safetyNote && (
                    <Stack direction="row" spacing={0.75} sx={{ mt: 3 }}>
                        <InfoOutlinedIcon sx={{ fontSize: 16, color: 'text.secondary', mt: '2px' }} />
                        <Typography variant="body2" color="textSecondary" fontStyle="italic">
                            {safetyNote}
                        </Typography>
                    </Stack>
                )}
            </DialogContent>
            <DialogActions sx={{ gap: 1, alignItems: 'flex-start', flexWrap: 'wrap' }}>
                {getSummary && draft.strategy && (
                    <Typography
                        variant="caption" color="textSecondary"
                        sx={{
                            flexGrow: 1, flexBasis: 200, px: 1, py: '9px', minWidth: 0,
                            // The summary can contain a user-typed pattern with no spaces (e.g. a long
                            // glob), which the browser can't find a natural break in — without this,
                            // that text overflows straight through the flex box and mashes into the
                            // buttons instead of wrapping, regardless of flexGrow/minWidth already set.
                            overflowWrap: 'anywhere', wordBreak: 'break-word',
                        }}
                    >
                        {getSummary(draft)}
                    </Typography>
                )}
                <Stack direction="row" spacing={1} sx={{ flexShrink: 0, ml: 'auto' }}>
                    <Button color="inherit" onClick={onClose} sx={{ whiteSpace: 'nowrap' }}>Cancel</Button>
                    <Button
                        variant="outlined" color="primary" disabled={!draft.strategy} onClick={() => onSave(draft)}
                        sx={{ whiteSpace: 'nowrap' }}
                    >
                        {isNew ? 'Add rule' : 'Apply'}
                    </Button>
                </Stack>
            </DialogActions>
        </Dialog>
    );
}

export default CleanupRuleDialog;
