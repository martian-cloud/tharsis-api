import { Alert, Box, Checkbox, Chip, FormControlLabel, Paper, Stack, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { MutationError } from '../common/error';
import Link from '../routes/Link';
import TagForm from './TagForm';
import { RunnerSettingsForm_runnerTags$key } from './__generated__/RunnerSettingsForm_runnerTags.graphql';

export interface FormData {
    inherit: boolean;
    tags: readonly string[];
}

interface Props {
    formData: FormData;
    fragmentRef: RunnerSettingsForm_runnerTags$key;
    error: MutationError | undefined;
    onChange: (data: FormData) => void;
    showInheritOption?: boolean;
}

function RunnerSettingsForm({ formData, fragmentRef, error, onChange, showInheritOption }: Props) {

    const data = useFragment<RunnerSettingsForm_runnerTags$key>(
        graphql`
        fragment RunnerSettingsForm_runnerTags on NamespaceRunnerTags
        {
            inherited
            namespacePath
            value
        }
        `, fragmentRef
    );

    const handleInherit = (event: React.ChangeEvent<HTMLInputElement>) => {
        const inherit = event.target.checked;
        if (data.inherited) {
            onChange({ tags: [], inherit });
        } else {
            onChange({ tags: formData.tags, inherit });
        }
    };

    return (
        <Box>
            {error && <Alert sx={{ mb: 2 }} severity={error.severity}>
                {error.message}
            </Alert>}
            <Typography variant="subtitle1" gutterBottom>Tags</Typography>
            <Typography variant="body2" mb={1} color="textSecondary">
                Use tags to target specific runners. The tags will be automatically added to any jobs created within this namespace.
            </Typography>
            {!showInheritOption && <Box>
                <FormControlLabel
                    control={<Checkbox
                        color="secondary"
                        checked={formData.inherit}
                        onChange={handleInherit}
                    />}
                    label="Inherit tags from parent group"
                />
                {formData.inherit && data.inherited && <Paper sx={{ p: 2 }}>
                    {data.value.length > 0 && <>
                        <Box display="flex" gap={0.5}>
                            <Typography variant='body2' color='textSecondary'>The following tags inherited from group</Typography>
                            <Link
                                to={`/groups/${data.namespacePath}`}
                                variant='body2' color='secondary'>
                                {data.namespacePath}
                            </Link>
                        </Box>
                        <Stack sx={{ mt: 2 }} direction="row" spacing={1}>
                            {data.value.map(tag => <Chip key={tag} size="small" color="default" label={tag} />)}
                        </Stack>
                    </>}
                    {data.value.length === 0 && <Typography variant='body2' color='textSecondary'>No tags set in parent group</Typography>}
                </Paper>}
            </Box>}
            {!formData.inherit &&
                <TagForm
                    data={formData.tags}
                    onChange={(data) => onChange({ ...formData, tags: data.tags })}
                />
            }
        </Box>
    );
}

export default RunnerSettingsForm;
