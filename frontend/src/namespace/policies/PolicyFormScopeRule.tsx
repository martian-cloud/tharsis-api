import DeleteIcon from '@mui/icons-material/Delete';
import { Box, FormControl, IconButton, InputLabel, MenuItem, Select, TextField } from '@mui/material';
import { SCOPE_TYPE_OPTIONS, ScopeRuleActionValue, ScopeRuleFormData, ScopeRuleTypeValue } from './scopeRules';

interface Props {
    rule: ScopeRuleFormData;
    onChange: (rule: ScopeRuleFormData) => void;
    onDelete: () => void;
    disableDelete?: boolean;
}

/**
 * PolicyFormScopeRule is one scope rule, edited in place. There is no separate step that commits the row
 * into the policy: every control writes straight back through onChange, so a filled-in row is a rule.
 */
function PolicyFormScopeRule({ rule, onChange, onDelete, disableDelete }: Props) {
    const selectedScopeType = SCOPE_TYPE_OPTIONS.find(o => o.value === rule.type);

    return (
        <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', alignItems: 'flex-start' }}>
            <FormControl size="small" sx={{ minWidth: 120 }}>
                <InputLabel>Action</InputLabel>
                <Select
                    label="Action"
                    value={rule.action}
                    onChange={e => onChange({ ...rule, action: e.target.value as ScopeRuleActionValue })}
                >
                    <MenuItem value="INCLUDE">Include</MenuItem>
                    <MenuItem value="EXCLUDE">Exclude</MenuItem>
                </Select>
            </FormControl>
            <FormControl size="small" sx={{ minWidth: 190 }}>
                <InputLabel>Type</InputLabel>
                <Select
                    label="Type"
                    value={rule.type}
                    onChange={e => onChange({ ...rule, type: e.target.value as ScopeRuleTypeValue })}
                >
                    {SCOPE_TYPE_OPTIONS.map(opt => (
                        <MenuItem key={opt.value} value={opt.value}>{opt.label}</MenuItem>
                    ))}
                </Select>
            </FormControl>
            <Box sx={{ flex: 1, minWidth: 220 }}>
                {/* The path carries over when the type changes: both types are paths, and switching
                    between them is how you say "match this against the identity instead". */}
                <TextField
                    size="small"
                    fullWidth
                    label="Path pattern"
                    placeholder={selectedScopeType?.placeholder ?? ''}
                    value={rule.pattern}
                    onChange={e => onChange({ ...rule, pattern: e.target.value })}
                />
            </Box>
            <IconButton
                size="small"
                onClick={onDelete}
                disabled={disableDelete}
                title="Delete rule"
                sx={{ alignSelf: 'center' }}
            >
                <DeleteIcon fontSize="small" />
            </IconButton>
        </Box>
    );
}

export default PolicyFormScopeRule;
