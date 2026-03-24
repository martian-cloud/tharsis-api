import { Alert, Box, Divider, FormControl, InputLabel, MenuItem, Select, Typography } from '@mui/material';
import { MutationError } from '../../common/error';
import LabelManager from '../../workspace/labels/LabelManager';
import { Label } from '../../workspace/labels/types';

export interface FormData {
    private: boolean
    labels: Label[]
}

interface Props {
    data: FormData
    onChange: (data: FormData) => void
    editMode?: boolean
    error?: MutationError
}

function TerraformModuleForm({ data, onChange, error }: Props) {
    return (
        <Box>
            {error && <Alert sx={{ marginTop: 2 }} severity={error.severity}>
                {error.message}
            </Alert>}
            <Typography sx={{ marginTop: 2 }} variant="subtitle1" gutterBottom>Visibility</Typography>
            <Divider sx={{ opacity: 0.6 }} />
            <Box sx={{ my: 2 }}>
                <FormControl size="small" sx={{ minWidth: 160 }}>
                    <InputLabel>Visibility</InputLabel>
                    <Select
                        label="Visibility"
                        value={data.private ? 'private' : 'internal'}
                        onChange={event => onChange({ ...data, private: event.target.value === 'private' })}
                    >
                        <MenuItem value="internal">Internal</MenuItem>
                        <MenuItem value="private">Private</MenuItem>
                    </Select>
                </FormControl>
                <Typography variant="caption" color="textSecondary" display="block" sx={{ mt: 1 }}>
                    Internal modules are accessible to all groups. Private modules are restricted to its parent group and subgroups.
                </Typography>
            </Box>
            <Box sx={{ my: 2 }}>
                <LabelManager
                    labels={data.labels}
                    onSave={labels => { onChange({ ...data, labels }); return Promise.resolve(); }}
                    title="Manage Module Labels"
                    description="Add, edit, or remove labels for this module. Labels are key-value pairs that help with organization and filtering."
                />
            </Box>
        </Box>
    );
}

export default TerraformModuleForm;
