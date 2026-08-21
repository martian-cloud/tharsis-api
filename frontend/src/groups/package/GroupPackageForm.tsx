import { Alert, Box, Divider, FormControl, FormControlLabel, InputLabel, MenuItem, Select, Switch, TextField, Typography } from '@mui/material';
import { MutationError } from '../../common/error';

export type PackageVisibility = 'PRIVATE' | 'ROOT_GROUP' | 'GLOBAL';
export type PackageKind = '' | 'OPA_POLICY';

export interface FormData {
    name: string
    description: string
    kind: PackageKind
    visibility: PackageVisibility
    allowMutableVersions: boolean
}

interface Props {
    data: FormData
    onChange: (data: FormData) => void
    editMode?: boolean
    error?: MutationError
}

// KIND_OPTIONS lists the supported package types. Only OPA is available today; the field is required
// so the type is always chosen explicitly.
const KIND_OPTIONS: { value: PackageKind, label: string, description: string }[] = [
    { value: 'OPA_POLICY', label: 'OPA', description: 'Open Policy Agent (Rego) policy bundle.' },
];

const VISIBILITY_OPTIONS: { value: PackageVisibility, label: string, description: string }[] = [
    { value: 'PRIVATE', label: 'Private', description: 'Restricted to the parent group and its subgroups.' },
    { value: 'ROOT_GROUP', label: 'Root Group', description: 'Available throughout the root group hierarchy.' },
    { value: 'GLOBAL', label: 'Global', description: 'Available to all groups.' },
];

function GroupPackageForm({ data, onChange, editMode, error }: Props) {
    return (
        <Box>
            {error && <Alert sx={{ marginTop: 2 }} severity={error.severity}>
                {error.message}
            </Alert>}
            <Box sx={{ my: 2 }}>
                <TextField
                    size="small"
                    fullWidth
                    label="Name"
                    disabled={editMode}
                    value={data.name}
                    onChange={event => onChange({ ...data, name: event.target.value })}
                />
            </Box>
            <Box sx={{ my: 2 }}>
                <TextField
                    size="small"
                    fullWidth
                    multiline
                    rows={3}
                    label="Description"
                    value={data.description}
                    onChange={event => onChange({ ...data, description: event.target.value })}
                />
            </Box>
            <Box sx={{ my: 2 }}>
                <FormControl size="small" sx={{ minWidth: 200 }} required disabled={editMode}>
                    <InputLabel>Type</InputLabel>
                    <Select
                        label="Type"
                        value={data.kind}
                        onChange={event => onChange({ ...data, kind: event.target.value as PackageKind })}
                    >
                        {KIND_OPTIONS.map(opt => (
                            <MenuItem key={opt.value} value={opt.value}>{opt.label}</MenuItem>
                        ))}
                    </Select>
                </FormControl>
                <Typography variant="caption" color="textSecondary" display="block" sx={{ mt: 1 }}>
                    {KIND_OPTIONS.find(o => o.value === data.kind)?.description ?? 'Select the package type.'}
                </Typography>
            </Box>
            <Typography sx={{ marginTop: 2 }} variant="subtitle1" gutterBottom>Visibility</Typography>
            <Divider sx={{ opacity: 0.6 }} />
            <Box sx={{ my: 2 }}>
                <FormControl size="small" sx={{ minWidth: 200 }}>
                    <InputLabel>Visibility</InputLabel>
                    <Select
                        label="Visibility"
                        value={data.visibility}
                        onChange={event => onChange({ ...data, visibility: event.target.value as PackageVisibility })}
                    >
                        {VISIBILITY_OPTIONS.map(opt => (
                            <MenuItem key={opt.value} value={opt.value}>{opt.label}</MenuItem>
                        ))}
                    </Select>
                </FormControl>
                <Typography variant="caption" color="textSecondary" display="block" sx={{ mt: 1 }}>
                    {VISIBILITY_OPTIONS.find(o => o.value === data.visibility)?.description}
                </Typography>
            </Box>
            <Typography sx={{ marginTop: 2 }} variant="subtitle1" gutterBottom>Versions</Typography>
            <Divider sx={{ opacity: 0.6 }} />
            <Box sx={{ my: 2 }}>
                <FormControlLabel
                    control={
                        <Switch
                            checked={data.allowMutableVersions}
                            onChange={event => onChange({ ...data, allowMutableVersions: event.target.checked })}
                        />
                    }
                    label="Allow mutable versions"
                />
                <Typography variant="caption" color="textSecondary" display="block" sx={{ mt: 1 }}>
                    When enabled, an already-uploaded version can be re-uploaded in place to replace its
                    contents without publishing a new version. When disabled, versions are immutable once uploaded.
                </Typography>
            </Box>
        </Box>
    );
}

export default GroupPackageForm;
