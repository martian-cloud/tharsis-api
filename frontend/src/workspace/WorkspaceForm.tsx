import { Alert, Box, Divider, TextField, Typography } from '@mui/material';
import { MutationError } from '../common/error';
import LabelManager from './labels/LabelManager';
import { Label } from './labels/types';

export interface FormData {
    name: string
    description: string
    fullPath?: string
    labels?: Label[]
}

interface Props {
    data: FormData
    onChange: (data: FormData) => void
    editMode?: boolean
    error?: MutationError
}

function WorkspaceForm({
    data,
    onChange,
    editMode,
    error,
}: Props) {

    const handleLabelsChange = async (labels: Label[]) => {
        // For new workspace creation, update form data
        onChange({ ...data, labels });
    };

    return (
        <Box>
            {error && <Alert sx={{ mb: 2 }} severity={error.severity}>
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

            <Divider light sx={{ my: 3 }} />
            <LabelManager
                labels={data.labels || []}
                onSave={handleLabelsChange}
                title="Workspace Labels"
                description="Add labels to categorize and organize this workspace. Labels help with filtering and identification."
            />
        </Box>
    );
}

export default WorkspaceForm;
