import { Alert, Box, Checkbox, Chip, FormControlLabel, Paper, Switch, Typography } from '@mui/material';
import { MutationError } from '../common/error';
import { useFragment } from 'react-relay/hooks';
import graphql from 'babel-plugin-relay/macro';
import Link from '../routes/Link';
import { DriftDetectionSettingsFormFragment_driftDetectionEnabled$key } from './__generated__/DriftDetectionSettingsFormFragment_driftDetectionEnabled.graphql';

export interface FormData {
    inherit: boolean;
    enabled: boolean;
}

interface Props {
    formData: FormData;
    onChange: (data: FormData) => void;
    error: MutationError | undefined;
    showInheritOption?: boolean;
    fragmentRef: DriftDetectionSettingsFormFragment_driftDetectionEnabled$key;
}

function DriftDetectionSettingsForm({ formData, onChange, error, showInheritOption, fragmentRef }: Props) {

    const data = useFragment<DriftDetectionSettingsFormFragment_driftDetectionEnabled$key>(
        graphql`
            fragment DriftDetectionSettingsFormFragment_driftDetectionEnabled on NamespaceDriftDetectionEnabled
            {
                inherited
                namespacePath
                value
            }
        `,
        fragmentRef
    );

    const onInheritChanged = (event: React.ChangeEvent<HTMLInputElement>) => {
        const inherit = event.target.checked;
        const enabled = inherit ? data.value : formData.enabled;

        onChange({ inherit, enabled });
    };

    return (
        <Box>
            {error && <Alert sx={{ mb: 2 }} severity={error.severity}>
                {error.message}
            </Alert>}
            <Typography
                sx={{ mb: 1, pr: 8 }}
                variant="body2"
                color="textSecondary"
            >
                When enabled, drift detection checks if your infrastructure's actual state differs from your defined configuration.
                You can either inherit these settings from the parent group or configure them specifically for this namespace.
            </Typography>
            {!showInheritOption && <Box sx={{ mt: 2, mb: 2 }}>
                <FormControlLabel
                    control={
                        <Checkbox
                            color="secondary"
                            checked={formData.inherit}
                            onChange={onInheritChanged}
                        />
                    }
                    label="Inherit from parent group"
                />
                {formData.inherit && data.inherited && <Paper sx={{ p: 2 }}>
                    <Box display="flex" gap={0.5}>
                        <Typography variant="body2" color="textSecondary">Drift detection inherited from group</Typography>
                        <Link
                            to={`/groups/${data.namespacePath}`}
                            variant="body2" color="secondary">
                            {data.namespacePath}
                        </Link>
                    </Box>
                    <Chip
                        sx={{ mt: 2 }}
                        size="xs"
                        color={data.value ? "secondary" : "info"}
                        label={data.value ? 'Enabled' : 'Disabled'}
                        variant="outlined"
                    />
                </Paper>}
            </Box>}
            {!formData.inherit &&
                <Box sx={{ mt: 2 }}>
                    <Typography variant="subtitle1">Automatic Drift Detection</Typography>
                    <FormControlLabel
                        control={
                            <Switch
                                sx={{ m: 2 }}
                                checked={formData.enabled}
                                color="secondary"
                                onChange={(e) => onChange({ ...formData, enabled: e.target.checked })}
                            />
                        }
                        label={formData.enabled ? 'Enabled' : 'Disabled'}
                    />
                </Box>
            }
        </Box>
    );
}

export default DriftDetectionSettingsForm;
