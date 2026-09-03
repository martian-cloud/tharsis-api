import { Box, Typography } from '@mui/material';
import PanelButton from '../../common/PanelButton';
import { PolicyStage } from './PolicyForm';

interface StageOption {
    value: PolicyStage;
    label: string;
    description: string;
}

interface Props {
    stage: PolicyStage;
    options: StageOption[];
    // The grid always sizes to every stage the backend supports, not just the ones currently
    // offered: module attestation's narrower option list would otherwise widen its columns relative
    // to OPA's, and the layout would shift if either kind's option list changes.
    columnCount: number;
    onSelect: (stage: PolicyStage) => void;
}

/**
 * StageSelector is the stage picker shared by every policy kind's configuration section: a row of
 * PanelButtons, one per stage the chosen kind's data can evaluate at (see stageOptionsForKind).
 * Kind-specific behavior on selection — OPA's post-apply advisory clamp — is the caller's
 * responsibility via onSelect, so this component stays the same for every kind.
 */
function StageSelector({ stage, options, columnCount, onSelect }: Props) {
    return (
        <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: `repeat(${columnCount}, 1fr)` }, gap: 2 }}>
            {options.map(opt => (
                <PanelButton
                    key={opt.value}
                    selected={stage === opt.value}
                    sx={{ minWidth: 0, maxWidth: 'none' }}
                    onClick={() => onSelect(opt.value)}
                >
                    <Typography variant="subtitle1">{opt.label}</Typography>
                    <Typography variant="caption" align="center">
                        {opt.description}
                    </Typography>
                </PanelButton>
            ))}
        </Box>
    );
}

export default StageSelector;
