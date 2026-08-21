import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import { Box, Typography, useTheme } from '@mui/material';
import { alpha } from '@mui/material/styles';

interface Props {
    // The failure messages across the check, one entry per violation.
    messages: readonly string[];
    // Set when messages leaves some of what the check reported out — it comes from the check's capped
    // summary, and a policy set producing a great many violations will overflow it.
    truncated?: boolean;
}

// RunTaskStagePolicyErrorsBox collects the failure messages a check reported, so a reader of the run's policy
// stage gets the whole picture without expanding the individual policies below it.
//
// It carries no heading: the passed/failed counts sit directly above it, which is label enough, and
// the error tint says what kind of block this is. The approvals inbox shows the same messages but has
// diverged into its own component — a card in a list needs a collapsed summary and cannot afford a red
// field on every row. See ApprovalGateFindings.
function RunTaskStagePolicyErrorsBox({ messages, truncated }: Props) {
    const theme = useTheme();

    if (messages.length === 0) {
        return null;
    }

    return (
        <Box
            sx={{
                mt: 2,
                // Error-tinted like the stage's other callouts — the same fill and border the overridden
                // box below it uses with warning.
                background: alpha(theme.palette.error.main, 0.09),
                borderRadius: '6px',
                padding: '14px 16px',
                display: 'flex',
                flexDirection: 'column',
                gap: 1.5,
            }}
        >
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.75 }}>
                {messages.map((message, i) => (
                    <Box key={i} sx={{ display: 'flex', gap: '7px', alignItems: 'flex-start' }}>
                        {/* mt centres the glyph on the first line of the message rather than on the
                            whole block, which is what a multi-line finding would otherwise do to it. */}
                        <ErrorOutlineIcon sx={{ width: 14, height: 14, mt: '3px', flexShrink: 0, color: theme.palette.error.main }} />
                        <Typography variant="body2" sx={{ whiteSpace: 'pre-wrap' }}>{message}</Typography>
                    </Box>
                ))}
            </Box>
            {truncated && (
                <Typography variant="caption" component="div" sx={{ color: theme.palette.text.secondary }}>
                    Too many errors to list them all here — each policy below shows its own in full.
                </Typography>
            )}
        </Box>
    );
}

export default RunTaskStagePolicyErrorsBox;
