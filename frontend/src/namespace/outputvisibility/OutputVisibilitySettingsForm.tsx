import { Alert, Box, Checkbox, Chip, FormControl, FormControlLabel, MenuItem, Paper, Select, Typography } from '@mui/material';
import { MutationError } from '../../common/error';
import { useFragment } from 'react-relay/hooks';
import graphql from 'babel-plugin-relay/macro';
import Link from '../../routes/Link';
import { OutputVisibilitySettingsFormFragment_outputVisibility$key, NamespaceOutputVisibilityLevel } from './__generated__/OutputVisibilitySettingsFormFragment_outputVisibility.graphql';

// Exclude Relay's "%future added value" sentinel from the type for our maps
type KnownVisibility = Exclude<NamespaceOutputVisibilityLevel, '%future added value'>;

const KNOWN_VISIBILITIES: Set<string> = new Set<string>([
    'block_access',
    'direct_group_only',
    'direct_group_and_subgroups',
    'root_group',
    'global',
]);

/** Safely cast a visibility value, falling back to BLOCK_ACCESS for unknown future enum members. */
export function toKnownVisibility(value: string): KnownVisibility {
    return KNOWN_VISIBILITIES.has(value) ? value as KnownVisibility : 'block_access';
}

const VISIBILITY_LABELS: Record<KnownVisibility, string> = {
    block_access: 'Block access',
    direct_group_only: 'Workspaces in this group',
    direct_group_and_subgroups: 'Workspaces in this group or child groups',
    root_group: 'Any workspace in root group',
    global: 'Any workspace in Tharsis',
};

const VISIBILITY_DESCRIPTIONS: Record<KnownVisibility, string> = {
    block_access: 'No workspace can read this workspace\'s outputs',
    direct_group_only: 'Only workspaces in the same immediate parent group can read outputs',
    direct_group_and_subgroups: 'Any workspace within the parent group\'s subtree (same group + descendant subgroups) can read outputs',
    root_group: 'Any workspace sharing the same root group can read outputs',
    global: 'Any workspace in the system can read outputs',
};

export interface FormData {
    inherit: boolean;
    visibility: KnownVisibility;
}

interface Props {
    formData: FormData;
    onChange: (data: FormData) => void;
    error: MutationError | undefined;
    isRootGroup?: boolean;
    fragmentRef: OutputVisibilitySettingsFormFragment_outputVisibility$key;
}

function OutputVisibilitySettingsForm({ formData, onChange, error, isRootGroup, fragmentRef }: Props) {

    const data = useFragment<OutputVisibilitySettingsFormFragment_outputVisibility$key>(
        graphql`
            fragment OutputVisibilitySettingsFormFragment_outputVisibility on NamespaceOutputVisibility
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
        const visibility = inherit ? toKnownVisibility(data.value) : formData.visibility;

        onChange({ inherit, visibility });
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
                Controls which workspaces can consume outputs from workspaces in this namespace via the <code>tharsis_workspace_outputs</code> data source.
                You can either inherit these settings from the parent group or configure them specifically for this namespace.
            </Typography>
            {!isRootGroup && <Box sx={{ mt: 2, mb: 2 }}>
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
                        <Typography variant="body2" color="textSecondary">Output visibility inherited from group</Typography>
                        <Link
                            to={`/groups/${data.namespacePath}`}
                            variant="body2" color="secondary">
                            {data.namespacePath}
                        </Link>
                    </Box>
                    <Chip
                        sx={{ mt: 2 }}
                        size="small"
                        color="info"
                        label={VISIBILITY_LABELS[data.value as KnownVisibility] ?? data.value}
                        variant="outlined"
                    />
                </Paper>}
            </Box>}
            {!formData.inherit &&
                <Box sx={{ mt: 2 }}>
                    <Typography variant="subtitle1" sx={{ mb: 1 }} id="output-visibility-label">Output Visibility</Typography>
                    <FormControl sx={{ minWidth: 300 }}>
                        <Select
                            size="small"
                            value={formData.visibility}
                            aria-labelledby="output-visibility-label"
                            onChange={(e) => onChange({ ...formData, visibility: e.target.value as KnownVisibility })}
                        >
                            {Object.entries(VISIBILITY_LABELS).map(([value, label]) => (
                                <MenuItem key={value} value={value}>
                                    {label}
                                </MenuItem>
                            ))}
                        </Select>
                    </FormControl>
                    <Typography
                        sx={{ mt: 1 }}
                        variant="body2"
                        color="textSecondary"
                    >
                        {VISIBILITY_DESCRIPTIONS[formData.visibility]}
                    </Typography>
                </Box>
            }
        </Box>
    );
}

export default OutputVisibilitySettingsForm;
