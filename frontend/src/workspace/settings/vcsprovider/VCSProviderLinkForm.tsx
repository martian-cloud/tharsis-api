import { Autocomplete, Box, Chip, Switch, TextField, Typography } from "@mui/material";

export interface VCSFormData {
    repositoryPath: string
    branch: string
    moduleDirectory: string
    tagRegex: string
    globPatterns: string[]
    autoSpeculativePlan: boolean
    webhookDisabled: boolean
}

export const EMPTY_VCS_FORM_DATA: VCSFormData = {
    repositoryPath: '',
    branch: '',
    moduleDirectory: '',
    tagRegex: '',
    globPatterns: [],
    autoSpeculativePlan: false,
    webhookDisabled: false,
};

interface LinkFields {
    readonly repositoryPath: string
    readonly branch: string
    readonly moduleDirectory: string | null | undefined
    readonly tagRegex: string | null | undefined
    readonly globPatterns: readonly string[]
    readonly autoSpeculativePlan: boolean
    readonly webhookDisabled: boolean
}

export function mapLinkToFormData(link: LinkFields): VCSFormData {
    return {
        repositoryPath: link.repositoryPath,
        branch: link.branch,
        moduleDirectory: link.moduleDirectory ?? '',
        tagRegex: link.tagRegex ?? '',
        globPatterns: [...link.globPatterns],
        autoSpeculativePlan: link.autoSpeculativePlan,
        webhookDisabled: link.webhookDisabled,
    };
}

export function isFormDataEqual(a: VCSFormData, b: VCSFormData): boolean {
    return a.repositoryPath === b.repositoryPath &&
        a.branch === b.branch &&
        a.moduleDirectory === b.moduleDirectory &&
        a.tagRegex === b.tagRegex &&
        a.autoSpeculativePlan === b.autoSpeculativePlan &&
        a.webhookDisabled === b.webhookDisabled &&
        a.globPatterns.length === b.globPatterns.length &&
        a.globPatterns.every((glob, index) => glob === b.globPatterns[index]);
}

interface Props {
    data: VCSFormData
    onChange: (data: VCSFormData) => void
    // The repository can't be changed on an existing link, only when creating one.
    repositoryPathReadOnly?: boolean
    repositoryPathError?: boolean
}

interface SwitchRowProps {
    label: string
    description: string
    checked: boolean
    onChange: (checked: boolean) => void
}

function SwitchRow({ label, description, checked, onChange }: SwitchRowProps) {
    return (
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 2 }}>
            <Box sx={{ minWidth: 0 }}>
                <Typography>{label}</Typography>
                <Typography variant="body2" color="textSecondary">{description}</Typography>
            </Box>
            <Switch
                checked={checked}
                color="secondary"
                onChange={event => onChange(event.target.checked)}
                slotProps={{ input: { 'aria-label': label } }}
                sx={{ flexShrink: 0 }}
            />
        </Box>
    );
}

function VCSProviderLinkForm({ data, onChange, repositoryPathReadOnly, repositoryPathError }: Props) {
    const onGlobPatternsChange = (values: string[]) => {
        const trimmed = values.map(value => value.trim()).filter(value => value !== '');
        onChange({ ...data, globPatterns: Array.from(new Set(trimmed)) });
    };

    return (
        <Box>
            <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' }, columnGap: 2, rowGap: 2 }}>
                <TextField
                    size="small"
                    fullWidth
                    label="Repository path"
                    value={data.repositoryPath}
                    disabled={repositoryPathReadOnly}
                    error={repositoryPathError}
                    helperText={repositoryPathError ? 'Repository path is required.' : 'Owner and repository, for example org/infra-modules.'}
                    onChange={event => onChange({ ...data, repositoryPath: event.target.value })}
                />
                <TextField
                    size="small"
                    fullWidth
                    label="Branch"
                    value={data.branch}
                    helperText="Pushes to this branch start runs. Defaults to the repository's default branch."
                    onChange={event => onChange({ ...data, branch: event.target.value })}
                />
                <TextField
                    size="small"
                    fullWidth
                    label="Module directory"
                    value={data.moduleDirectory}
                    helperText="Folder Terraform runs from. Defaults to the repository root."
                    onChange={event => onChange({ ...data, moduleDirectory: event.target.value })}
                />
                <TextField
                    size="small"
                    fullWidth
                    label="Tag regex"
                    value={data.tagRegex}
                    helperText="Tags that match start runs. If empty, tags are ignored."
                    onChange={event => onChange({ ...data, tagRegex: event.target.value })}
                    slotProps={{ input: { sx: { typography: 'code' } } }}
                />
                <Autocomplete
                    multiple
                    freeSolo
                    autoSelect
                    options={[]}
                    value={data.globPatterns}
                    onChange={(event, values) => onGlobPatternsChange(values as string[])}
                    sx={{ gridColumn: '1 / -1' }}
                    renderValue={(values, getItemProps) => values.map((value, index) => {
                        const { key, ...itemProps } = getItemProps({ index });
                        return <Chip key={key} size="small" variant="outlined" label={value} sx={{ typography: 'code' }} {...itemProps} />;
                    })}
                    renderInput={params => (
                        <TextField
                            {...params}
                            size="small"
                            label="Glob patterns"
                            placeholder="Type a pattern, press Enter"
                            helperText="Only changes to matching files start runs. If empty, any change starts a run."
                        />
                    )}
                />
            </Box>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 3 }}>
                <SwitchRow
                    label="Speculative Plans"
                    description="Run a speculative plan whenever a merge request is opened or updated. Speculative plans preview changes and can't be applied."
                    checked={data.autoSpeculativePlan}
                    onChange={checked => onChange({ ...data, autoSpeculativePlan: checked })}
                />
                <SwitchRow
                    label="Webhook Push Events"
                    description="Start runs automatically when the repository receives pushes or merge request updates. Turn off to ignore these events."
                    checked={!data.webhookDisabled}
                    onChange={checked => onChange({ ...data, webhookDisabled: !checked })}
                />
            </Box>
        </Box>
    );
}

export default VCSProviderLinkForm
