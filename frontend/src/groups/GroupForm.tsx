import { Alert, Box, Divider, TextField, Typography } from '@mui/material';
import { MutationError } from '../common/error';

export interface FormData {
    name: string
    description: string
    fullPath?: string
}

interface Props {
    data: FormData
    onChange: (data: FormData) => void
    editMode?: boolean
    error?: MutationError
}

function GroupForm({ data, onChange, editMode, error }: Props) {

    return (
        <Box>
            {error && <Alert sx={{ marginBottom: 2 }} severity={error.severity}>
                {error.message}
            </Alert>}
            <Typography variant="subtitle1" gutterBottom>Details</Typography>
            <Divider light />
            <Box marginTop={2} marginBottom={2}>
                <TextField
                    disabled={editMode}
                    size="small"
                    fullWidth
                    label="Name"
                    value={data.name}
                    onChange={event => onChange({ ...data, name: event.target.value })}
                />
                <TextField
                    size="small"
                    margin='normal'
                    fullWidth
                    label="Description"
                    value={data.description}
                    onChange={event => onChange({ ...data, description: event.target.value })}
                />
            </Box>
        </Box>
    );
}

export default GroupForm
