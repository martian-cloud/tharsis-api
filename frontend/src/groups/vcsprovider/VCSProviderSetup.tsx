import { Box, Divider, Stack, IconButton, Paper, TextField, Typography } from '@mui/material'
import CopyIcon from '@mui/icons-material/ContentCopy';
import { FormData } from './VCSProviderForm';
import { StyledCode } from '../../common/StyledCode';
import cfg from '../../common/config';

interface Props {
    data: FormData
    onChange: (data: FormData) => void
}

function VCSProviderSetup({ data, onChange }: Props) {

    return (
        <Box>
            {(data.type) ? <Box>
                <Typography sx={{ mt: 2, mb: 2 }} variant="h5" gutterBottom>Provider Setup</Typography>
                <Typography>Start the process to register a new OAuth application in {data.type ===  'github' ? 'GitHub' : 'GitLab'}. Copy the following, required for creating a new OAuth application:</Typography>
                <Stack sx={{ mt: 2 , mb: 2, ml: 2 }}>
                {data.type === 'github' ?
                    <Stack direction="row" marginBottom={0.5}>
                        <Typography variant="body1" component="span"><strong>Homepage URL:</strong> {window.location.protocol}//{window.location.host}</Typography>
                        <IconButton sx={{ padding: '4px' }} onClick={() => navigator.clipboard.writeText(`${window.location.protocol}//${window.location.host}`)}>
                            <CopyIcon sx={{ width: 16, height: 16 }} />
                        </IconButton>
                    </Stack> : null}
                    <Stack direction="row">
                        <Typography variant="body1" component="span" sx={{ fontWeight: 'bold' }}>{data.type === 'github' ? 'Callback URL' : 'Redirect URI'}:
                            <Typography variant="body1" component="span">{` ${cfg.apiUrl}/v1/vcs/auth/callback`}</Typography>
                        </Typography>
                        <IconButton sx={{ padding: '4px' }} onClick={() => navigator.clipboard.writeText(`${cfg.apiUrl}/v1/vcs/auth/callback`)}>
                            <CopyIcon sx={{ width: 16, height: 16 }} />
                        </IconButton>
                    </Stack>
                </Stack>
            {data.type === 'gitlab' ? <Box marginBottom={2}>
                <Typography>Enable the <StyledCode>Confidential</StyledCode> setting. Additionally, enable the following two scopes:</Typography>
                <Paper sx={{ pt: 0.25, pb: 0.25, mt: 1, mb: 1 }}>
                    <ul>
                        <li>{data.autoCreateWebhooks ? 'api' : 'read_api'}</li>
                        <li>{data.autoCreateWebhooks ? 'read_repository' : 'read_user'}</li>
                    </ul>
                </Paper>
            </Box> : null}
            <Box>
                <Typography>After registering a new OAuth application, {data.type === 'github' ? 'a Client ID and Client Secret' : 'an Application ID and Secret value'} will be generated. Copy and paste the ID and Secret into the fields below.</Typography>
            </Box>
            </Box> : null}
            {data.type && <Box marginTop={2} marginBottom={2}>
                <TextField
                    size="small"
                    fullWidth
                    label={data.type === 'github' ? "Client ID" : "Application ID"}
                    value={data.oAuthClientId}
                    onChange={event => onChange({ ...data, oAuthClientId: event.target.value })}
                />
                <TextField
                    size="small"
                    margin="normal"
                    fullWidth
                    label={(data.type === 'github' ? "Client Secret" : "Secret")}
                    value={data.oAuthClientSecret}
                    onChange={event => onChange({ ...data, oAuthClientSecret: event.target.value })}
                />
            </Box>}
            {data.type && <Box marginBottom={2}>
                <Typography>
                    If creation is successful, Tharsis will immediately generate a new authorization URL and redirect the browser to the VCS provider to finalize the OAuth flow.
                </Typography>
            </Box>}
            <Divider light/>
        </Box>
    );
}

export default VCSProviderSetup
