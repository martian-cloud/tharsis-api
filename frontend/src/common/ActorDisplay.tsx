import Avatar from '@mui/material/Avatar';
import Box from '@mui/material/Box';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import Link from '../routes/Link';
import Gravatar from './Gravatar';
import { JobIcon, TharsisIcon, UnavailableResourceIcon, UserIcon, VCSProviderIcon } from './Icons';
import { ActorDisplayFragment_actor$key } from './__generated__/ActorDisplayFragment_actor.graphql';

// ActorFields are the values an actor is rendered from. They mirror the Actor GraphQL type: trn is the
// principal's TRN and is set only for service accounts, VCS providers and jobs, and a user is shown
// from its principal's email because users never carry a TRN.
export interface ActorFields {
    type: string;
    id?: string | null;
    trn?: string | null;
    userEmail?: string | null;
}

export type ActorAvatarKind = 'gravatar' | 'initial' | 'job' | 'vcsProvider' | 'system' | 'scim' | 'deletedUser' | 'unknown';

export interface ActorDescription {
    text: string;
    avatar: ActorAvatarKind;
    // email is set when avatar is 'gravatar'.
    email?: string;
    // initial is set when avatar is 'initial'.
    initial?: string;
    link?: string;
}

const parentPath = (path: string): string => path.substring(0, path.lastIndexOf('/'));

const lastSegmentInitial = (path: string): string => path.charAt(path.lastIndexOf('/') + 1).toUpperCase();

// trnPath returns the resource path a TRN ("trn:<type>:<path>") names, or the value unchanged when it
// is not a TRN.
export const trnPath = (value: string): string => {
    const match = /^trn:[^:]+:(.+)$/.exec(value);
    return match ? match[1] : value;
};

// describeActor decides what to show for an actor.
export function describeActor(actor: ActorFields | null | undefined): ActorDescription | null {
    if (!actor) {
        return null;
    }

    const { type, id, userEmail } = actor;
    // The TRN is shown by the resource path it names.
    const path = actor.trn ? trnPath(actor.trn) : null;

    switch (type) {
        case 'USER':
            if (userEmail) {
                return { text: userEmail, avatar: 'gravatar', email: userEmail };
            }
            // Any caller can view every user, so a user actor without its user was deleted.
            return { text: 'Deleted user', avatar: 'deletedUser' };
        case 'SERVICE_ACCOUNT':
            if (path) {
                return {
                    text: path,
                    avatar: 'initial',
                    initial: lastSegmentInitial(path),
                    link: id ? `/groups/${parentPath(path)}/-/service_accounts/${id}` : undefined,
                };
            }
            return { text: 'Unknown service account', avatar: 'unknown' };
        case 'VCS_PROVIDER':
            if (path) {
                return {
                    text: path,
                    avatar: 'vcsProvider',
                    link: id ? `/groups/${parentPath(path)}/-/vcs_providers/${id}` : undefined,
                };
            }
            return { text: 'Unknown VCS provider', avatar: 'vcsProvider' };
        case 'JOB':
            if (path) {
                // A job's path is its workspace path followed by the job's ID.
                const workspacePath = parentPath(path);
                return {
                    text: `Job in ${workspacePath}`,
                    avatar: 'job',
                    link: id ? `/groups/${workspacePath}/-/jobs/${id}` : undefined,
                };
            }
            return { text: 'Unknown job', avatar: 'job' };
        case 'SYSTEM':
            return { text: 'System', avatar: 'system' };
        case 'SCIM':
            return { text: 'SCIM', avatar: 'scim' };
        default:
            return null;
    }
}

export type ActorDisplayVariant = 'avatar' | 'inline' | 'text';

function ActorAvatar({ description, size }: { description: ActorDescription, size: number }) {
    const iconSx = { width: size, height: size, bgcolor: 'avatar.default', color: 'text.primary' };
    const iconFontSize = { fontSize: size * 0.7 };

    switch (description.avatar) {
        case 'gravatar':
            return <Gravatar width={size} height={size} email={description.email as string} />;
        case 'initial':
            return (
                <Avatar
                    variant="rounded"
                    sx={{
                        width: size,
                        height: size,
                        bgcolor: 'avatar.default',
                        fontSize: theme => theme.typography.body2.fontSize,
                        fontWeight: 500
                    }}>
                    {description.initial}
                </Avatar>
            );
        case 'job':
            return <Avatar variant="rounded" sx={iconSx}><JobIcon sx={iconFontSize} /></Avatar>;
        case 'vcsProvider':
            return <Avatar variant="rounded" sx={iconSx}><VCSProviderIcon sx={iconFontSize} /></Avatar>;
        case 'system':
            return <Avatar variant="rounded" sx={iconSx}><TharsisIcon sx={iconFontSize} /></Avatar>;
        case 'scim':
            return <Avatar sx={iconSx}><UserIcon sx={iconFontSize} /></Avatar>;
        case 'deletedUser':
            return <Avatar sx={{ ...iconSx, bgcolor: 'action.disabledBackground', color: 'text.disabled' }}><UserIcon sx={iconFontSize} /></Avatar>;
        default:
            return <Avatar sx={iconSx}><UnavailableResourceIcon sx={iconFontSize} /></Avatar>;
    }
}

interface Props {
    fragmentRef: ActorDisplayFragment_actor$key | null | undefined;
    // avatar shows only the avatar with the actor in a tooltip, inline shows the avatar followed by
    // the actor, and text shows only the actor.
    variant?: ActorDisplayVariant;
    size?: number;
    // placeholder is shown when there is no actor.
    placeholder?: string;
}

// ActorDisplay renders the principal that performed an action. Service accounts, VCS providers and
// jobs are shown from the TRN snapshot recorded with the action, so they need no further lookup; only a
// user is shown from its user record, because a user never carries a TRN. Each actor type is a
// separate GraphQL type, so the API looks up a user only for user actors.
function ActorDisplay({ fragmentRef, variant = 'avatar', size = 20, placeholder = '--' }: Props) {
    const data = useFragment<ActorDisplayFragment_actor$key>(graphql`
        fragment ActorDisplayFragment_actor on Actor {
            type
            principalId
            ... on UserActor {
                user {
                    email
                }
            }
            ... on ServiceAccountActor {
                trnSnapshot
            }
            ... on VCSProviderActor {
                trnSnapshot
            }
            ... on JobActor {
                trnSnapshot
            }
        }
    `, fragmentRef);

    const description = describeActor(data ? { type: data.type, id: data.principalId, trn: data.trnSnapshot, userEmail: data.user?.email } : null);

    if (!description) {
        return variant === 'avatar' ? null : <Typography component="span" variant="inherit" color="textSecondary">{placeholder}</Typography>;
    }

    const text = description.link
        ? <Link color="inherit" to={description.link} sx={{ wordBreak: 'break-word' }}>{description.text}</Link>
        : <Box component="span" sx={{ wordBreak: 'break-word' }}>{description.text}</Box>;

    switch (variant) {
        case 'text':
            return text;
        case 'inline':
            return (
                <Box component="span" sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5, verticalAlign: 'middle', minWidth: 0 }}>
                    <ActorAvatar description={description} size={size} />
                    {text}
                </Box>
            );
        default:
            return (
                <Tooltip title={description.text}>
                    <Box sx={{ display: 'inline-flex' }}>
                        <ActorAvatar description={description} size={size} />
                    </Box>
                </Tooltip>
            );
    }
}

export default ActorDisplay;
