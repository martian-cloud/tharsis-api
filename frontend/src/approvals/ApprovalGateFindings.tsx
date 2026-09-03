import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import { Box, Typography, useTheme } from '@mui/material';
import { darken } from '@mui/material/styles';

interface Props {
    // The failure messages across the check, one entry per violation.
    messages: readonly string[];
    // Set when messages leaves some of what the check reported out — it comes from the check's capped
    // summary, and a policy set producing a great many violations will overflow it.
    truncated?: boolean;
}

// ApprovalGateFindings lists what a gated check reported, on the card for one gate in the approvals
// inbox: a labelled section of the card, not a panel within it. The findings are why the gate is here,
// so they sit in the card's own surface rather than behind a border that would read as a second card.
//
// The run's policy stage shows the same messages as a tinted callout, where it is the one such block on
// the page and has the per-policy cards below it to stand apart from; see RunTaskStagePolicyErrorsBox.
function ApprovalGateFindings({ messages, truncated }: Props) {
    const theme = useTheme();

    if (messages.length === 0) {
        return null;
    }

    return (
        <Box
            sx={{
                mt: 2,
                p: 1,
                display: 'flex',
                flexDirection: 'column',
                gap: 1,
                background: darken(theme.palette.background.paper, 0.10)
            }}
        >
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.75 }}>
                {messages.map((message, i) => (
                    <Box key={i} sx={{ display: 'flex', gap: '7px', alignItems: 'flex-start' }}>
                        {/* mt centres the glyph on the first line of the message rather than on the whole
                            block. Neutral, not error-coloured: one row per finding would be a lot of red
                            for a card that is only ever shown because something failed. */}
                        <ErrorOutlineIcon sx={{ width: 14, height: 14, mt: '3px', flexShrink: 0, color: theme.palette.text.secondary }} />
                        <Typography variant="body2" color="textSecondary" sx={{ whiteSpace: 'pre-wrap' }}>{message}</Typography>
                    </Box>
                ))}
            </Box>
            {truncated && (
                <Typography variant="caption" color="warning">
                    Too many errors to list them all here — open the run's policy stage to see each policy's
                    findings in full.
                </Typography>
            )}
        </Box>
    );
}

export default ApprovalGateFindings;
