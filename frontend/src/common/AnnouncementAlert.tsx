import { Alert, AlertColor, IconButton } from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import { AnnouncementType } from './__generated__/AnnouncementBannerQuery.graphql';
import Markdown from './Markdown';

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
    return (
        <Alert
            severity={typeToSeverity[type]}
            variant="filled"
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
                // Override info alert color - theme uses transparent white
                ...(type === 'INFO' && {
                    backgroundColor: '#1976d2 !important',
                }),
            }}
        >
            <Markdown>{message}</Markdown>
        </Alert>
    );
}

export default AnnouncementAlert;
