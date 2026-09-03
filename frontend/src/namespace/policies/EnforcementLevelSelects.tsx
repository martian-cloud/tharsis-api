import { FormControl, InputLabel, ListItemText, MenuItem, Select, Typography, Box } from '@mui/material';
import { ENFORCEMENT_OPTIONS, SPECULATIVE_ENFORCEMENT_OPTIONS } from './PolicyForm';
import type { PolicyEnforcementLevel, SpeculativeRunEnforcementLevel } from './PolicyForm';

interface Props {
    enforcementLevel: PolicyEnforcementLevel;
    speculativeRunEnforcementLevel: SpeculativeRunEnforcementLevel;
    onEnforcementLevelChange: (level: PolicyEnforcementLevel) => void;
    onSpeculativeRunEnforcementLevelChange: (level: SpeculativeRunEnforcementLevel) => void;
    // Only OPA's post-apply stage clamps both levels to advisory (state has already been written by
    // the time a post-apply check evaluates, so there is no run outcome left for a stronger level to
    // protect — see models.Policy.Validate on the backend). Module attestation never offers that
    // stage (see stageOptionsForKind), so it has nothing to clamp and this defaults to false.
    disabled?: boolean;
}

/**
 * EnforcementLevelSelects is the pair of enforcement-level dropdowns shared by every policy kind's
 * configuration section: one for apply runs, one for speculative plans and assessment runs (which
 * have no soft-mandatory option — an override there would unblock nothing, since nobody is waiting
 * on either to approve one). The clamp to advisory when disabled is the caller's responsibility, via
 * the values it passes in and the onChange handlers it supplies — this component only renders and
 * disables, so it stays the same for every kind.
 */
function EnforcementLevelSelects({
    enforcementLevel,
    speculativeRunEnforcementLevel,
    onEnforcementLevelChange,
    onSpeculativeRunEnforcementLevelChange,
    disabled = false,
}: Props) {
    return (
        <>
            {disabled && <Typography variant="caption" color="textSecondary" display="block" sx={{ mb: 1 }}>
                Post-apply policies can only be enforced at Advisory: state has already been written by the time this stage evaluates, so there is no run outcome left for a stronger level to block.
            </Typography>}
            <Box sx={{ display: 'flex', gap: 3, flexWrap: 'wrap', mb: 1 }}>
                <FormControl size="small" sx={{ width: { xs: '100%', sm: 420 } }} disabled={disabled}>
                    <InputLabel>Apply Runs</InputLabel>
                    <Select
                        label="Apply Runs"
                        value={enforcementLevel}
                        // renderValue overrides what the closed control shows: without it, Select
                        // renders the chosen MenuItem's full children (label + description) in the
                        // closed, single-line control, which is what was clipping the description.
                        // The description still shows in full in the open dropdown below.
                        renderValue={value => ENFORCEMENT_OPTIONS.find(o => o.value === value)?.label ?? value}
                        onChange={event => onEnforcementLevelChange(event.target.value as PolicyEnforcementLevel)}
                    >
                        {ENFORCEMENT_OPTIONS.map(opt => (
                            <MenuItem key={opt.value} value={opt.value}>
                                <ListItemText primary={opt.label} secondary={opt.description} />
                            </MenuItem>
                        ))}
                    </Select>
                </FormControl>
                <FormControl size="small" sx={{ width: { xs: '100%', sm: 420 } }} disabled={disabled}>
                    <InputLabel>Speculative & Assessment Runs</InputLabel>
                    <Select
                        label="Speculative & Assessment Runs"
                        value={speculativeRunEnforcementLevel}
                        renderValue={value => SPECULATIVE_ENFORCEMENT_OPTIONS.find(o => o.value === value)?.label ?? value}
                        onChange={event => onSpeculativeRunEnforcementLevelChange(event.target.value as SpeculativeRunEnforcementLevel)}
                    >
                        {SPECULATIVE_ENFORCEMENT_OPTIONS.map(opt => (
                            <MenuItem key={opt.value} value={opt.value}>
                                <ListItemText primary={opt.label} secondary={opt.description} />
                            </MenuItem>
                        ))}
                    </Select>
                </FormControl>
            </Box>
        </>
    );
}

export default EnforcementLevelSelects;
