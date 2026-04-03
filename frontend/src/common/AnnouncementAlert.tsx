import { Alert, AlertColor, IconButton } from '@mui/material';
import { useTheme } from '@mui/material/styles';
import CloseIcon from '@mui/icons-material/Close';
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome';
import { AnnouncementType } from './__generated__/AnnouncementBannerQuery.graphql';
import Markdown from './Markdown';

const typeToAnnouncementKey = {
    INFO: 'info',
    ERROR: 'error',
    WARNING: 'warning',
    SUCCESS: 'success',
} as const;

const typeToSeverity: Record<string, AlertColor> = {
    INFO: 'info',
    ERROR: 'error',
    WARNING: 'warning',
    SUCCESS: 'success'
};

interface Props {
    id: string;
    message: string;
    type: AnnouncementType;
    dismissible: boolean;
    onDismiss?: (id: string) => void;
    disabled?: boolean;
}

function AnnouncementAlert({ id, message, type, dismissible, onDismiss, disabled }: Props) {
    const theme = useTheme();
    const key = typeToAnnouncementKey[type as keyof typeof typeToAnnouncementKey] ?? 'info';
    const p = theme.palette.announcement[key];
    const gradient = `linear-gradient(135deg, ${p.main}, ${p.dark}, ${p.light}, ${p.main})`;

    const multiLine = message.includes('\n');

    return (
        <Alert
            severity={typeToSeverity[type]}
            variant="filled"
            icon={type === 'SUCCESS' ? <AutoAwesomeIcon /> : undefined}
            action={
                dismissible ? (
                    <IconButton
                        aria-label="dismiss"
                        color="inherit"
                        size="small"
                        onClick={() => onDismiss?.(id)}
                        disabled={disabled}
                    >
                        <CloseIcon fontSize="inherit" />
                    </IconButton>
                ) : null
            }
            sx={{
                background: gradient,
                backgroundSize: '200% 200%',
                animation: 'gradientShift 3s ease forwards',
                color: theme.palette.common.white,
                '& .MuiAlert-icon': { color: theme.palette.common.white, alignItems: multiLine ? 'flex-start' : 'center' },
                '& .MuiAlert-message': { fontSize: '0.85rem', fontWeight: 700, '& p': { fontSize: 'inherit', fontWeight: 'inherit' }, '& a': { color: 'inherit', textDecoration: 'none', fontWeight: 900 } },
                '@keyframes gradientShift': {
                    '0%': { backgroundPosition: '0% 50%' },
                    '100%': { backgroundPosition: '100% 50%' },
                },
            }}
        >
            <Markdown>{message}</Markdown>
        </Alert>
    );
}

export default AnnouncementAlert;
