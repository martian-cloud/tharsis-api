import { Alert, Box, Button, Dialog, DialogActions, DialogContent, DialogTitle, IconButton, InputAdornment, TextField, Tooltip, Typography, useMediaQuery, useTheme } from '@mui/material';
import VisibilityIcon from '@mui/icons-material/Visibility';
import VisibilityOffIcon from '@mui/icons-material/VisibilityOff';
import { useState } from 'react';
import CopyButton from '../../../common/CopyButton';
import { providerLabel } from './utils';

export type ManualWebhookContext = 'new' | 'reset' | 'provider_change';

export interface ManualWebhookDetails {
    context: ManualWebhookContext
    url: string
    // Only GitLab takes a separate secret token; GitHub's token is part of the URL.
    token: string | null | undefined
    providerType: string
    repositoryPath: string
}

interface Props {
    details: ManualWebhookDetails
    onClose: () => void
}

const TITLES: Record<ManualWebhookContext, (provider: string) => string> = {
    new: provider => `Add the webhook to ${provider}`,
    reset: provider => `Update the webhook in ${provider}`,
    provider_change: provider => `Add the new webhook in ${provider}`,
};

function buildSteps(details: ManualWebhookDetails): string[] {
    const { repositoryPath, providerType, context } = details;

    if (providerType === 'gitlab') {
        return [
            context === 'reset'
                ? `Open ${repositoryPath}, then go to Settings → Webhooks and edit the Tharsis webhook.`
                : `Open ${repositoryPath}, then go to Settings → Webhooks → Add new webhook.`,
            'URL: paste the webhook URL.',
            'Secret token: paste the token.',
            'Trigger: tick Push events, Tag push events and Merge request events.',
            `Keep Enable SSL verification on, then select ${context === 'reset' ? 'Save changes' : 'Add webhook'}.`,
        ];
    }

    return [
        context === 'reset'
            ? `Open ${repositoryPath}, then go to Settings → Webhooks and edit the Tharsis webhook.`
            : `Open ${repositoryPath}, then go to Settings → Webhooks → Add webhook.`,
        'Payload URL: paste the webhook URL.',
        'Content type: application/json.',
        'Events: choose individual events, then tick Pushes and Pull requests.',
        `Leave Active ticked and select ${context === 'reset' ? 'Update webhook' : 'Add webhook'}.`,
    ];
}

function ManualWebhookDialog({ details, onClose }: Props) {
    const [showToken, setShowToken] = useState(false);
    const theme = useTheme();
    const fullScreen = useMediaQuery(theme.breakpoints.down('sm'));

    const provider = providerLabel(details.providerType);
    const hasToken = details.providerType === 'gitlab' && !!details.token;
    const secretNoun = hasToken ? 'token' : 'URL';

    const warning = details.context === 'reset'
        ? `The old webhook has stopped working. Paste ${hasToken ? 'these' : 'this'} into ${details.repositoryPath} to start runs again. Tharsis won't show the ${secretNoun} again.`
        : `Copy the ${secretNoun} now. Tharsis won't show it again.`;

    return (
        <Dialog open maxWidth="sm" fullWidth fullScreen={fullScreen}>
            <DialogTitle>{TITLES[details.context](provider)}</DialogTitle>
            <DialogContent dividers>
                <Alert severity="warning" sx={{ mb: 3 }}>{warning}</Alert>
                <TextField
                    fullWidth
                    size="small"
                    label="Webhook URL"
                    value={details.url}
                    sx={{ mb: 2 }}
                    slotProps={{
                        input: {
                            readOnly: true,
                            sx: { typography: 'code' },
                            endAdornment: (
                                <InputAdornment position="end">
                                    <CopyButton data={details.url} toolTip="Copy URL" />
                                </InputAdornment>
                            )
                        }
                    }}
                />
                {hasToken && (
                    <TextField
                        fullWidth
                        size="small"
                        label="Token"
                        type={showToken ? 'text' : 'password'}
                        autoComplete="off"
                        value={details.token}
                        sx={{ mb: 2 }}
                        slotProps={{
                            input: {
                                readOnly: true,
                                sx: { typography: 'code' },
                                endAdornment: (
                                    <InputAdornment position="end">
                                        <Tooltip title={showToken ? 'Hide token' : 'Show token'}>
                                            <IconButton
                                                size="small"
                                                aria-label={showToken ? 'Hide token' : 'Show token'}
                                                onClick={() => setShowToken(prev => !prev)}
                                            >
                                                {showToken ? <VisibilityOffIcon fontSize="small" /> : <VisibilityIcon fontSize="small" />}
                                            </IconButton>
                                        </Tooltip>
                                        <CopyButton data={details.token ?? ''} toolTip="Copy token" />
                                    </InputAdornment>
                                )
                            }
                        }}
                    />
                )}
                <Typography variant="subtitle2" sx={{ mt: 1, mb: 1 }}>In {provider}</Typography>
                <Box component="ol" sx={{ m: 0, pl: 3 }}>
                    {buildSteps(details).map(step => (
                        <Typography component="li" variant="body2" key={step} sx={{ mb: 0.5 }}>{step}</Typography>
                    ))}
                </Box>
            </DialogContent>
            <DialogActions>
                <Button variant="contained" onClick={onClose}>Done</Button>
            </DialogActions>
        </Dialog>
    );
}

export default ManualWebhookDialog;
