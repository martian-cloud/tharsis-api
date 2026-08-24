import LockOutlinedIcon from '@mui/icons-material/LockOutlined';
import { Box, LinearProgress, Typography, useTheme } from '@mui/material';
import { alpha } from '@mui/material/styles';

interface Props {
    // Failed policies an approval rule governs. Progress is counted in policies rather than
    // decisions: a policy is cleared once its rule has collected the approvals it requires.
    gatedCount: number;
    approvedCount: number;
}

// RunTaskStageOverrideProgressBox reports how far a gate's approvals have got. Shared by the run's policy stage
// panel and the approvals inbox card, which show the same block.
function RunTaskStageOverrideProgressBox({ gatedCount, approvedCount }: Props) {
    const theme = useTheme();

    if (gatedCount === 0) {
        return null;
    }

    const percent = (approvedCount / gatedCount) * 100;

    return (
        <Box
            sx={{
                mt: 2,
                // Neutral rather than warning-tinted: this is a progress readout that sits alongside
                // the policies it counts, so it should not compete with them for attention.
                background: alpha(theme.palette.common.white, 0.03),
                borderRadius: '6px',
                padding: '14px 16px',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                gap: 2.5,
                flexWrap: 'wrap',
            }}
        >
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
                <LockOutlinedIcon sx={{ width: 18, height: 18, color: theme.palette.warning.main, flexShrink: 0 }} />
                <Box>
                    <Typography variant="subtitle2" component="div">
                        {gatedCount} polic{gatedCount === 1 ? 'y' : 'ies'} waiting approval to override
                    </Typography>
                    <Typography variant="body2" sx={{ color: theme.palette.text.secondary }}>
                        {approvedCount} of {gatedCount} policies approved
                    </Typography>
                </Box>
            </Box>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, minWidth: 180, flex: '1 1 180px' }}>
                <LinearProgress
                    variant="determinate"
                    value={percent}
                    sx={{
                        flex: 1,
                        minWidth: 120,
                        height: 6,
                        borderRadius: '9999px',
                        backgroundColor: alpha(theme.palette.common.white, 0.12),
                        '& .MuiLinearProgress-bar': { backgroundColor: theme.palette.text.secondary },
                    }}
                />
                <Typography variant="body2" sx={{ fontWeight: 600, fontVariantNumeric: 'tabular-nums', color: theme.palette.text.secondary }}>
                    {approvedCount} / {gatedCount}
                </Typography>
            </Box>
        </Box>
    );
}

export default RunTaskStageOverrideProgressBox;
