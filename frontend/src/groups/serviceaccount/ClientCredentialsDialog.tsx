import { Button, Dialog, DialogActions, DialogContent, DialogTitle, InputAdornment, TextField, Typography } from '@mui/material';
import CopyButton from '../../common/CopyButton';
import cfg from '../../common/config';

interface Props {
    clientId: string;
    clientSecret: string;
    onClose: () => void;
}

function ClientCredentialsDialog({ clientId, clientSecret, onClose }: Props) {
    const tokenEndpoint = `${cfg.apiUrl}/v1/serviceaccounts/token`;

    const copyAdornment = (data: string) => (
        <InputAdornment position="end">
            <CopyButton data={data} toolTip="Click to copy" />
        </InputAdornment>
    );

    return (
        <Dialog open maxWidth="md" fullWidth>
            <DialogTitle>Client Credentials</DialogTitle>
            <DialogContent>
                <Typography variant="body2" color="warning.main" sx={{ mb: 1 }}>
                    <strong>Save these credentials now.</strong> The client secret will not be shown again.
                </Typography>
                <TextField
                    fullWidth
                    size="small"
                    label="Client ID"
                    value={clientId}
                    margin="normal"
                    slotProps={{ input: { readOnly: true, endAdornment: copyAdornment(clientId) } }}
                />
                <TextField
                    fullWidth
                    size="small"
                    label="Client Secret"
                    value={clientSecret}
                    margin="normal"
                    slotProps={{ input: { readOnly: true, endAdornment: copyAdornment(clientSecret) } }}
                />
                <TextField
                    fullWidth
                    size="small"
                    label="Token Endpoint (POST)"
                    value={tokenEndpoint}
                    margin="normal"
                    slotProps={{ input: { readOnly: true, endAdornment: copyAdornment(tokenEndpoint) } }}
                />
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>Done</Button>
            </DialogActions>
        </Dialog>
    );
}

export default ClientCredentialsDialog;
