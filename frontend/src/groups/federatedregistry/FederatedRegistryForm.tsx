import { Alert, Box, Divider, TextField, Typography, SxProps, Theme } from '@mui/material';
import { StyledCode } from '../../common/StyledCode';
import { MutationError } from '../../common/error';

export interface FormData {
    hostname: string;
    audience: string;
}

interface Props {
    data: FormData;
    onChange: (data: FormData) => void;
    error?: MutationError;
    sx?: SxProps<Theme>;
}

function FederatedRegistryForm({ data, onChange, error, sx }: Props) {
    return (
        <Box sx={sx}>
            {error && <Alert sx={{ mb: 2 }} severity={error.severity}>
                {error.message}
            </Alert>}
            <Typography variant="subtitle1" gutterBottom>Details</Typography>
            <Divider sx={{ opacity: 0.6 }} />
            <Box sx={{ my: 2 }}>
                <TextField
                    sx={{ mb: 1 }}
                    fullWidth
                    label="Hostname"
                    size="small"
                    value={data.hostname}
                    onChange={(e) => onChange({ ...data, hostname: e.target.value })}
                />
                <Typography
                    sx={{ mb: 2 }}
                    variant="subtitle2"
                    color="textSecondary"
                >
                    The network address of the federated registry &#x28;e.g.,{' '}
                    <StyledCode>registry.example.com</StyledCode>&#x29;.
                </Typography>
                <TextField
                    fullWidth
                    label="Audience"
                    size="small"
                    margin="normal"
                    value={data.audience}
                    onChange={(e) => onChange({ ...data, audience: e.target.value })}
                />
                <Typography
                    variant="subtitle2"
                    color="textSecondary"
                >
                    The audience will be added to the <StyledCode>aud</StyledCode> claim in the OIDC token used to authenticate with the federated registry. It must match the audience in the federated registry trust policy.
                </Typography>
            </Box>
        </Box>
    );
}

export default FederatedRegistryForm;
