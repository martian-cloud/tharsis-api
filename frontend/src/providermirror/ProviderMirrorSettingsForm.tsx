import { Alert, AlertTitle, Box, Checkbox, Chip, FormControlLabel, Paper, Switch, Typography } from '@mui/material';
import { MutationError } from '../common/error';
import { useFragment } from 'react-relay/hooks';
import graphql from 'babel-plugin-relay/macro';
import Link from '../routes/Link';
import { ProviderMirrorSettingsFormFragment_providerMirrorEnabled$key } from './__generated__/ProviderMirrorSettingsFormFragment_providerMirrorEnabled.graphql';

export interface FormData {
    inherit: boolean;
    enabled: boolean;
}

interface Props {
    formData: FormData;
    onChange: (data: FormData) => void;
    error: MutationError | undefined;
    showInheritOption?: boolean;
    fragmentRef: ProviderMirrorSettingsFormFragment_providerMirrorEnabled$key;
}

const LockFileWarning = () => (
    <Alert severity="info" variant="outlined" sx={{ mt: 2 }}>
        <AlertTitle>Expected Behavior</AlertTitle>
        Run logs may show <code style={{ color: 'inherit' }}><Box component="span" sx={{ color: 'warning.main' }}>Warning</Box>: Incomplete lock file information</code> — network mirrors only record checksums for the current platform. This is safe to ignore.
    </Alert>
);

function ProviderMirrorSettingsForm({ formData, onChange, error, showInheritOption, fragmentRef }: Props) {

    const data = useFragment<ProviderMirrorSettingsFormFragment_providerMirrorEnabled$key>(
        graphql`
            fragment ProviderMirrorSettingsFormFragment_providerMirrorEnabled on NamespaceProviderMirrorEnabled
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
                When enabled, external Terraform providers are automatically cached and verified during runs, providing resilience when upstream registries are unavailable.
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
                        <Typography variant="body2" color="textSecondary">Provider mirror inherited from group</Typography>
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
                    {data.value && <LockFileWarning />}
                </Paper>}
            </Box>}
            {!formData.inherit &&
                <Box sx={{ mt: 2 }}>
                    <Typography variant="subtitle1">Automatic Provider Caching</Typography>
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
                    {formData.enabled && <LockFileWarning />}
                </Box>
            }
        </Box>
    );
}

export default ProviderMirrorSettingsForm;
