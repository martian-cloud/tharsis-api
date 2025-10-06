import { Alert, Box } from '@mui/material'
import { MutationError } from '../../common/error';
import VCSProviderGeneralDetails from './VCSProviderGeneralDetails';
import VCSProviderSetup from './VCSProviderSetup'

export interface FormData {
    type: 'gitlab' | 'github' | undefined
    name: string
    description: string
    oAuthClientId: string
    oAuthClientSecret: string
    autoCreateWebhooks: boolean
    url: string
}

interface Props {
    data: FormData
    onChange: (data: FormData) => void
    editMode?: boolean
    error?: MutationError
}

function VCSProviderForm({ data, onChange, editMode, error }: Props) {

    return (
        <Box>
            <Box sx={{ mt: 2, mb: 2}}>
                {error && <Alert sx={{ mt: 2, mb: 2 }} severity={error.severity}>
                    {error.message}
                </Alert>}
                <VCSProviderGeneralDetails
                    data={data}
                    editMode={editMode}
                    onChange={(data: FormData) => onChange(data)}
                />
                {(!data.type || editMode) ? null : <VCSProviderSetup
                        data={data}
                        onChange={(data: FormData) => onChange(data)}
                    />}
            </Box>
        </Box>
    )
}

export default VCSProviderForm
