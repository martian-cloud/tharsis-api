import { Alert, Box, Divider, TextField, Typography } from '@mui/material'
import { MutationError } from '../../common/error';
import { VCSProviderType } from './__generated__/EditVCSProviderOAuthCredentialsQuery.graphql';

export interface OAuthFormData {
    type?: VCSProviderType
    oAuthClientId: string
    oAuthClientSecret: string
}

interface Props {
    data: OAuthFormData
    onChange: (data: OAuthFormData) => void
    error?: MutationError
}

function VCSProviderOAuthCredForm( { data, onChange, error }: Props) {

    return (
        <Box sx={{ mt: 2, mb: 2}}>
            {error && <Alert sx={{ mt: 2, mb: 2 }} severity={error.severity}>
                {error.message}
                </Alert>}
                <TextField
                    size="small"
                    fullWidth
                    label={data.type === "github" ?  "ClientID - write only" : "Application ID - write only"}
                    value={data.oAuthClientId}
                    onChange={event => onChange({ ...data, oAuthClientId: event.target.value })}
                />
                <TextField
                    size="small"
                    margin="normal"
                    fullWidth
                    label={data.type === "github" ? "Client Secret - write only" : "Secret - write only"}
                    value={data.oAuthClientSecret}
                    onChange={event => onChange({ ...data, oAuthClientSecret: event.target.value })}
                />
                <Box marginBottom={2}>
                    <Typography sx={{ mb: 2 }} variant="subtitle2" color="textSecondary">
                        After updating your OAuth credentials, you may need to reset your OAuth token.
                    </Typography>
                </Box>
            <Divider light/>
        </Box>
    )
}

export default VCSProviderOAuthCredForm
