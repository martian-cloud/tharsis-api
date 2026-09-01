import { useEffect, useState } from 'react';
import { Box, FormControlLabel, Radio, RadioGroup, Stack, TextField, Typography, useTheme } from '@mui/material';
import { alpha } from '@mui/material/styles';
import { CleanupRule, CleanupStrategy } from './types';

const STRATEGIES: Array<{ value: CleanupStrategy; label: string; detail: string }> = [
    { value: 'COUNT', label: 'Keep a fixed number', detail: 'Deletes everything past the newest few' },
    { value: 'AGE', label: 'Delete past an age', detail: 'Deletes anything older, keeping a floor' },
    { value: 'PROTECT', label: 'Never delete', detail: 'Shields matches from every rule below' },
];

interface Props {
    rule: CleanupRule;
    onChange: (rule: CleanupRule) => void;
    availableStrategies?: CleanupStrategy[];
}

// NumberField tracks its own raw input text rather than deriving it from `value` on every render
// (so the box can sit empty while retyping instead of snapping back to the last committed number).
function NumberField({ label, width, value, min = 1, onChange }: {
    label: string; width: number; value: number; min?: number; onChange: (value: number) => void;
}) {
    const [text, setText] = useState(String(value));

    // Re-sync when the value changes from outside this field (e.g. switching strategy resets it).
    useEffect(() => {
        setText(String(value));
    }, [value]);

    return (
        <TextField
            size="small" type="number" label={label}
            slotProps={{ htmlInput: { min } }}
            sx={{ width: { xs: '100%', sm: width }, flexGrow: { sm: 1 }, maxWidth: { sm: width * 1.6 } }}
            value={text}
            onChange={event => {
                const raw = event.target.value;
                setText(raw);
                if (raw !== '') onChange(Math.max(min, Number(raw) || min));
            }}
            onBlur={() => {
                // Leaving the field empty snaps back to the last valid value instead of staying blank.
                if (text === '') setText(String(value));
            }}
        />
    );
}

function CleanupRuleStrategyPicker({ rule, onChange, availableStrategies }: Props) {
    const theme = useTheme();
    const onStrategyChange = (strategy: CleanupStrategy) => {
        onChange({
            ...rule,
            strategy,
            ...('keepMin' in rule && { keepMin: strategy === 'PROTECT' ? 0 : Math.max(rule.keepMin as number, 1) }),
            deleteAfterDays: strategy === 'AGE' ? Math.max(rule.deleteAfterDays, 7) : 0,
        });
    };

    const visibleStrategies = availableStrategies
        ? STRATEGIES.filter(s => availableStrategies.includes(s.value))
        : STRATEGIES;

    return (
        <RadioGroup value={rule.strategy} onChange={event => onStrategyChange(event.target.value as CleanupStrategy)}>
            <Stack spacing={1}>
                {visibleStrategies.map(strategy => {
                    const selected = rule.strategy === strategy.value;
                    return (
                        <Box
                            key={strategy.value}
                            onClick={() => onStrategyChange(strategy.value)}
                            sx={{
                                display: 'flex', flexDirection: { xs: 'column', sm: 'row' },
                                alignItems: { xs: 'stretch', sm: 'center' }, justifyContent: 'space-between', gap: 1,
                                pl: 2, pr: 2, py: 1, borderRadius: 1, cursor: 'pointer',
                                border: '1px solid',
                                borderColor: selected ? 'primary.main' : 'divider',
                                bgcolor: selected ? alpha(theme.palette.primary.main, 0.16) : 'transparent',
                                '&:hover': { borderColor: selected ? 'primary.main' : 'text.disabled' },
                            }}
                        >
                            <FormControlLabel
                                sx={{ flexGrow: 1, ml: 0, mr: 0 }}
                                value={strategy.value}
                                control={<Radio color="primary" size="small" />}
                                label={
                                    <Box>
                                        <Typography variant="body2">{strategy.label}</Typography>
                                        <Typography variant="caption" color="textSecondary">{strategy.detail}</Typography>
                                    </Box>
                                }
                            />
                            {strategy.value === 'COUNT' && selected && 'keepMin' in rule && (
                                <NumberField
                                    label="Keep newest" width={120} value={rule.keepMin as number}
                                    onChange={keepMin => onChange({ ...rule, keepMin })}
                                />
                            )}
                            {strategy.value === 'AGE' && selected && (
                                <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1}>
                                    <NumberField
                                        label="Days" width={100} value={rule.deleteAfterDays} min={7}
                                        onChange={deleteAfterDays => onChange({ ...rule, deleteAfterDays })}
                                    />
                                    {'keepMin' in rule && (
                                        <NumberField
                                            label="Always keep" width={120} value={rule.keepMin as number}
                                            onChange={keepMin => onChange({ ...rule, keepMin })}
                                        />
                                    )}
                                </Stack>
                            )}
                        </Box>
                    );
                })}
            </Stack>
        </RadioGroup>
    );
}

export default CleanupRuleStrategyPicker;
