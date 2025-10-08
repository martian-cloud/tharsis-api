import { Alert, Box, Checkbox, Divider, FormControlLabel, Switch, TextField, Typography } from '@mui/material';
import { MutationError } from '../common/error';
import TagForm from '../runnertags/TagForm';

export interface FormData {
    name: string
    description: string
    disabled: boolean
    tags: string[]
    runUntaggedJobs: boolean
}

interface Props {
    data: FormData
    onChange: (data: FormData) => void
    editMode?: boolean
    error?: MutationError
}

function RunnerForm({ data, onChange, editMode, error }: Props) {

    return (
        <Box>
            {error && <Alert sx={{ mt: 2 }} severity={error.severity}>
                {error.message}
            </Alert>}
            <Typography sx={{ mt: 2 }} variant="subtitle1" gutterBottom>Details</Typography>
            <Divider light sx={{ mb: 2 }} />
            <Box sx={{ mt: 2, mb: 2 }}>
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
            <Typography sx={{ mt: 2 }} variant="subtitle1" gutterBottom>Tags</Typography>
            <Divider light sx={{ mb: 2 }} />
            <Box sx={{ mb: 3 }}>
                <Typography variant="body2" mb={1} color="textSecondary">
                    Add tags to target specific jobs. A runner must have all tags specified for a job in order to claim it.
                </Typography>
                <Box mb={2}>
                    <TagForm
                        data={data.tags}
                        onChange={({ tags }: { tags: string[] }) => onChange({ ...data, tags })}
                    />
                </Box>
                <FormControlLabel
                    control={<Checkbox
                        color="secondary"
                        checked={data.runUntaggedJobs}
                        onChange={event => onChange({ ...data, runUntaggedJobs: event.target.checked })}
                    />}
                    label="Run Untagged Jobs"
                />
            </Box>
            <Typography sx={{ mt: 2 }} variant="subtitle1" gutterBottom>Status</Typography>
            <Divider light sx={{ mb: 2 }} />
            <Box sx={{ mb: 4 }}>
                <Typography variant="body2" mb={1} color="textSecondary">
                    A runner will not claim any jobs when it's disabled.
                </Typography>
                <FormControlLabel
                    control={<Switch
                        sx={{ m: 2 }}
                        checked={!data.disabled}
                        color="secondary"
                        onChange={event => onChange({ ...data, disabled: !event.target.checked })}
                    />}
                    label={data.disabled ? "Disabled" : "Enabled"}
                />
                {!editMode && <Typography variant="subtitle2">
                    By default, the runner will be enabled when created. Check the switch to disable the runner. After the runner is created, it can be enabled and disabled as needed.
                </Typography>}
            </Box>
        </Box>
    );
}

export default RunnerForm
