import { Theme, Tooltip, useTheme } from '@mui/material';
import Pill from '../../common/Pill';

// Maps each EmailDeliveryStatus enum value to a label, a description surfaced in a tooltip, and a
// resolver for its semantic Pill color; a null color renders the neutral outline variant.
const STATUS_META: Record<string, { label: string; description: string; color: (theme: Theme) => string | null }> = {
    PENDING: { label: 'Pending', description: 'Queued and waiting to be sent.', color: () => null },
    ACCEPTED: { label: 'Accepted', description: 'Accepted by the email provider; awaiting a delivery result.', color: theme => theme.palette.info.main },
    COMPLETED: { label: 'Completed', description: 'Delivered, or accepted for delivery by providers that don\'t report delivery.', color: theme => theme.palette.success.main },
    SOFT_BOUNCED: { label: 'Soft Bounced', description: 'The recipient\'s mail server rejected the message temporarily (e.g. full mailbox); it may be retried.', color: theme => theme.palette.warning.main },
    DELAYED: { label: 'Delayed', description: 'The recipient\'s mail server is temporarily delaying delivery (e.g. throttling); it may still complete or bounce later.', color: theme => theme.palette.warning.main },
    HARD_BOUNCED: { label: 'Hard Bounced', description: 'The recipient\'s mail server rejected the message permanently (e.g. address doesn\'t exist); the address is suppressed.', color: theme => theme.palette.error.main },
    FAILED: { label: 'Failed', description: 'The provider rejected the message, or sending failed after exhausting retries; not retried.', color: theme => theme.palette.error.main },
    ABANDONED: { label: 'Abandoned', description: 'No delivery feedback arrived within the feedback window, so the message was given up on.', color: theme => theme.palette.warning.main },
};

interface Props {
    status: string;
}

function AdminAreaEmailDeliveryStatusChip({ status }: Props) {
    const theme = useTheme();
    const meta = STATUS_META[status] ?? { label: status, description: '', color: () => null };
    const color = meta.color(theme);

    const pill = color
        ? <Pill variant="tint" size="small" color={color}>{meta.label}</Pill>
        : <Pill variant="outline" size="small">{meta.label}</Pill>;

    return meta.description
        ? <Tooltip title={meta.description}><span>{pill}</span></Tooltip>
        : pill;
}

export default AdminAreaEmailDeliveryStatusChip;
