import { Theme, Tooltip, useTheme } from '@mui/material';
import Pill from '../../common/Pill';

// Maps each EmailSuppressionCause enum value to a label, a description surfaced in a tooltip, and a
// resolver for its semantic Pill color.
const CAUSE_META: Record<string, { label: string; description: string; color: (theme: Theme) => string }> = {
    HARD_BOUNCE: { label: 'Hard Bounce', description: 'A message to this address bounced permanently.', color: theme => theme.palette.error.main },
    COMPLAINT: { label: 'Complaint', description: 'The recipient marked a message as spam.', color: theme => theme.palette.warning.main },
};

interface Props {
    cause: string;
}

function AdminAreaEmailSuppressionCauseChip({ cause }: Props) {
    const theme = useTheme();
    const meta = CAUSE_META[cause] ?? { label: cause, description: '', color: (t: Theme) => t.palette.text.primary };

    const pill = <Pill variant="tint" size="small" color={meta.color(theme)}>{meta.label}</Pill>;

    return meta.description
        ? <Tooltip title={meta.description}><span>{pill}</span></Tooltip>
        : pill;
}

export default AdminAreaEmailSuppressionCauseChip;
