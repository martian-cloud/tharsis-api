import { Avatar, useTheme } from '@mui/material';
import { alpha } from '@mui/material/styles';
import Gravatar from '../../../common/Gravatar';

// The approver conventions from the design's "Approvers Options" sheet: a team is a rounded square,
// an individual is a circle. Service accounts are individuals but take the theme's distinct
// service-account tint so they don't read as users.
type PrincipalKind = 'user' | 'team' | 'serviceAccount';

// The size every principal avatar on the task stage screens renders at — the required-approver rows,
// the stacked "who approved" cluster, and the review rows. Kept here so they stay in step.
export const PRINCIPAL_AVATAR_SIZE = 22;

interface Props {
    kind: PrincipalKind;
    size: number;
    // An email for users (rendered as a Gravatar); otherwise the display name, whose first letter
    // becomes the initial.
    label: string;
}

function RunTaskStagePrincipalAvatar({ kind, size, label }: Props) {
    const theme = useTheme();

    if (kind === 'user') {
        return <Gravatar width={size} height={size} email={label} />;
    }

    const color = kind === 'team' ? theme.palette.avatar.default : theme.palette.avatar.serviceAccount;

    return (
        <Avatar
            variant={'rounded'}
            sx={{
                width: size,
                height: size,
                flexShrink: 0,
                bgcolor: alpha(color, 0.2),
                color,
                fontSize: size <= 26 ? '0.625rem' : theme.typography.caption.fontSize,
                fontWeight: 700,
                borderRadius: kind === 'team' ? '6px' : undefined,
            }}
        >
            {(label[0] ?? '?').toUpperCase()}
        </Avatar>
    );
}

export default RunTaskStagePrincipalAvatar;
