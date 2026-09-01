import { Box, Button } from '@mui/material';
import LoadingButton from '@mui/lab/LoadingButton';
import { Link as RouterLink } from 'react-router-dom';

interface Props {
    primaryLabel: string;
    onPrimaryClick: () => void;
    primaryDisabled?: boolean;
    // Shows a spinner in place of the label and disables the button — pass the in-flight state of
    // the create/update mutation so the button reflects that a save is actually happening.
    primaryLoading?: boolean;
    cancelDisabled?: boolean;
    // Both New and Edit cancel to the overview page one level up, but this stays overridable.
    cancelTo?: string;
}

// Action buttons (primary + Cancel) shared by the New and Edit cleanup policy forms. Rendered
// in normal document flow — no sticky/fixed positioning — so it scrolls along with the rest of
// the page instead of pinning to the viewport and overlapping whatever content sits above it.
function CleanupPolicyFormActions({
    primaryLabel, onPrimaryClick, primaryDisabled, primaryLoading, cancelDisabled, cancelTo = '..',
}: Props) {
    return (
        <Box
            sx={{
                mt: 1,
                py: 1.5,
                borderTop: '1px solid',
                borderColor: 'divider',
                display: 'flex',
                alignItems: 'center',
                gap: 3,
            }}
        >
            <LoadingButton
                variant="outlined" color="primary"
                disabled={primaryDisabled}
                loading={primaryLoading}
                onClick={onPrimaryClick}
                sx={{ whiteSpace: 'nowrap' }}
            >
                {primaryLabel}
            </LoadingButton>
            <Button
                component={RouterLink} to={cancelTo}
                disabled={cancelDisabled}
                sx={{ color: 'text.primary' }}
            >
                Cancel
            </Button>
        </Box>
    );
}

export default CleanupPolicyFormActions;
