import ArrowDownIcon from '@mui/icons-material/ArrowDownward';
import ArrowUpIcon from '@mui/icons-material/ArrowUpward';
import DeleteIcon from '@mui/icons-material/CloseOutlined';
import EditIcon from '@mui/icons-material/EditOutlined';
import { Box, IconButton, Stack, Typography } from '@mui/material';
import { ResponsiveRow, useCardMode } from '../../common/ResponsiveTable';
import NumberBadge from './NumberBadge';
import Verdict from './Verdict';
import { CleanupRule, Condition } from './types';
import ConditionText from './ConditionText';

// Action buttons are drawn as adjoining bordered squares (radius 0, borders collapsed via negative
// margin) rather than separated buttons, matching the design's button-group look.
const ACTION_BUTTON_SX = {
    border: '1px solid', borderColor: 'divider', borderRadius: 0,
    '&:not(:first-of-type)': { ml: '-1px' },
};

// ConditionBadge splits a condition's prose label from its literal value (a glob or a status), so
// the value reads as code rather than blending into the surrounding phrase.
function ConditionBadge({ label, value }: Condition) {
    return (
        <Stack
            direction="row" spacing={0.5} alignItems="center"
            sx={{ px: 1, py: 0.25, borderRadius: 1, border: '1px solid', borderColor: 'divider' }}
        >
            <ConditionText label={label} value={value} />
        </Stack>
    );
}

interface Props {
    rule: CleanupRule;
    resource: string;
    conditions: Condition[];
    // index is 1-based, shown as the rule's priority number.
    index: number;
    readOnly?: boolean;
    canMoveUp: boolean;
    canMoveDown: boolean;
    onEdit?: () => void;
    onMove?: (offset: number) => void;
    onDelete?: () => void;
}

// CleanupRuleListItem renders a single rule as a table row — the resource label and rule data are
// all that differs per kind, so this one component covers all kinds.
function CleanupRuleListItem({ rule, resource, conditions, index, readOnly, canMoveUp, canMoveDown, onEdit, onMove, onDelete }: Props) {
    const cardMode = useCardMode();
    // PROTECT rules never delete anything, so they read as the "safe" outcome; COUNT/AGE rules
    // delete something, so they read as the "will delete" outcome.
    const dotColor = rule.strategy === 'PROTECT' ? 'success.main' : 'warning.main';

    return (
        <ResponsiveRow cells={[
            ...(!cardMode ? [{ content: <NumberBadge value={index} /> }] : []),
            {
                primary: true,
                label: 'Outcome',
                content: (
                    <Box>
                        <Stack direction="row" alignItems="center" spacing={1}>
                            <Box sx={{ width: 8, height: 8, borderRadius: '50%', bgcolor: dotColor, flexShrink: 0 }} />
                            <Typography variant="body2" fontWeight={500}><Verdict rule={rule} resource={resource} /></Typography>
                        </Stack>
                        {rule.description && (
                            <Typography variant="caption" color="textSecondary">{rule.description}</Typography>
                        )}
                    </Box>
                ),
            },
            {
                label: 'Applies To',
                content: conditions.length > 0
                    ? (
                        <Stack direction="row" spacing={0.5} flexWrap="wrap" useFlexGap>
                            {conditions.map(condition => (
                                <ConditionBadge key={condition.label} {...condition} />
                            ))}
                        </Stack>
                    )
                    : <Typography variant="body2" color="textSecondary">every {resource}</Typography>,
            },
            ...(readOnly ? [] : [{
                footer: true,
                align: 'right' as const,
                content: (
                    <Stack direction="row" justifyContent="flex-end">
                        <IconButton size="small" disabled={!canMoveUp} aria-label="move rule up" onClick={() => onMove?.(-1)} sx={ACTION_BUTTON_SX}>
                            <ArrowUpIcon fontSize="small" />
                        </IconButton>
                        <IconButton size="small" disabled={!canMoveDown} aria-label="move rule down" onClick={() => onMove?.(1)} sx={ACTION_BUTTON_SX}>
                            <ArrowDownIcon fontSize="small" />
                        </IconButton>
                        <IconButton size="small" aria-label="edit rule" onClick={() => onEdit?.()} sx={ACTION_BUTTON_SX}>
                            <EditIcon fontSize="small" />
                        </IconButton>
                        <IconButton size="small" aria-label="delete rule" onClick={() => onDelete?.()} sx={ACTION_BUTTON_SX}>
                            <DeleteIcon fontSize="small" />
                        </IconButton>
                    </Stack>
                ),
            }]),
        ]} />
    );
}

export default CleanupRuleListItem;
