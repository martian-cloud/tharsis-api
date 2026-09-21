import { Chip, Tooltip } from '@mui/material';

// Tooltip copy for each outbox lifecycle state, kept here so the wording lives in one place.
const PERSISTENT_DESCRIPTION = 'Kept indefinitely for inspection.';

interface Props {
    ephemeral: boolean | null | undefined;
    // retentionDays is how long an ephemeral email is kept after sending before cleanup. Omitted in the
    // list view (which doesn't load config); when provided it's always a concrete day count, never null.
    retentionDays?: number;
}

function AdminAreaEmailOutboxLifecycleChip({ ephemeral, retentionDays }: Props) {
    const ephemeralDescription = retentionDays !== undefined
        ? `Kept for ${retentionDays} day${retentionDays === 1 ? '' : 's'} after sending, then automatically removed.`
        : 'Kept for a short retention window after sending, then automatically removed.';

    return (
        <Tooltip title={ephemeral ? ephemeralDescription : PERSISTENT_DESCRIPTION}>
            <Chip
                label={ephemeral ? 'Ephemeral' : 'Persistent'}
                color={ephemeral ? 'default' : 'info'}
                size="small"
            />
        </Tooltip>
    );
}

export default AdminAreaEmailOutboxLifecycleChip;
