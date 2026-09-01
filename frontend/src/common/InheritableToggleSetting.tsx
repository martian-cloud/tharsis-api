import { ReactNode, useMemo } from 'react';
import { Alert, Box, Button, Checkbox, Chip, FormControlLabel, Paper, Switch, Typography } from '@mui/material';
import { MutationError } from './error';
import Link from '../routes/Link';

export interface FormData {
    inherit: boolean;
    enabled: boolean;
}

interface Setting {
    inherited: boolean;
    namespacePath: string;
    value: boolean;
}

interface Props {
    namespacePath: string;
    setting: Setting;
    formData: FormData;
    onChange: (data: FormData) => void;
    onSave: () => void;
    isSaving: boolean;
    error?: MutationError;
    toggleTitle: string;
    inheritedLabel: string;
    inheritedLinkTo?: string;
    description?: ReactNode;
    enabledContent?: ReactNode;
}

function InheritableToggleSetting({
    namespacePath,
    setting,
    formData,
    onChange,
    onSave,
    isSaving,
    error,
    toggleTitle,
    inheritedLabel,
    inheritedLinkTo,
    description,
    enabledContent
}: Props) {
    // A root namespace has no parent to inherit from.
    const showInherit = namespacePath.includes('/');

    const noChanges = useMemo(
        () => setting.inherited === formData.inherit && setting.value === formData.enabled,
        [setting, formData]
    );

    // Snap the switch to the inherited value so it never shows a value that saving wouldn't keep.
    const onInheritChanged = (inherit: boolean) => {
        onChange({ inherit, enabled: inherit ? setting.value : formData.enabled });
    };

    return (
        <Box>
            {error && <Alert sx={{ mb: 2 }} severity={error.severity}>
                {error.message}
            </Alert>}
            {description && <Typography
                sx={{ mb: 1, pr: 8 }}
                variant="body2"
                color="textSecondary"
            >
                {description}
            </Typography>}
            {showInherit && <Box sx={{ mt: description ? 2 : 0, mb: 2 }}>
                <FormControlLabel
                    control={
                        <Checkbox
                            color="secondary"
                            checked={formData.inherit}
                            onChange={event => onInheritChanged(event.target.checked)}
                        />
                    }
                    label="Inherit from parent group"
                />
                {formData.inherit && setting.inherited && <Paper sx={{ p: 2 }}>
                    <Box display="flex" gap={0.5}>
                        <Typography variant="body2" color="textSecondary">{inheritedLabel}</Typography>
                        <Link
                            to={inheritedLinkTo ?? `/groups/${setting.namespacePath}`}
                            variant="body2" color="secondary">
                            {setting.namespacePath}
                        </Link>
                    </Box>
                    <Chip
                        sx={{ mt: 2 }}
                        size="xs"
                        color={setting.value ? 'secondary' : 'info'}
                        label={setting.value ? 'Enabled' : 'Disabled'}
                        variant="outlined"
                    />
                    {setting.value && enabledContent}
                </Paper>}
            </Box>}
            {!formData.inherit && <Box sx={{ mt: 2 }}>
                <Typography variant="subtitle1">{toggleTitle}</Typography>
                <FormControlLabel
                    control={
                        <Switch
                            sx={{ m: 2 }}
                            checked={formData.enabled}
                            color="secondary"
                            onChange={event => onChange({ ...formData, enabled: event.target.checked })}
                        />
                    }
                    label={formData.enabled ? 'Enabled' : 'Disabled'}
                />
                {formData.enabled && enabledContent}
            </Box>}
            <Box>
                <Button
                    sx={{ mt: 2 }}
                    size="small"
                    disabled={noChanges}
                    loading={isSaving}
                    variant="outlined"
                    color="primary"
                    onClick={onSave}
                >
                    Save changes
                </Button>
            </Box>
        </Box>
    );
}

export default InheritableToggleSetting;
