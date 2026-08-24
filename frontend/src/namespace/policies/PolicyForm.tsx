import AddIcon from '@mui/icons-material/Add';
import DeleteIcon from '@mui/icons-material/Delete';
import { Alert, Avatar, Box, Button, Divider, FormControl, IconButton, InputLabel, List, ListItem, ListItemText, MenuItem, Select, Stack, styled, TextField, Typography, useTheme } from '@mui/material';
import { useRef } from 'react';
import { MutationError } from '../../common/error';
import Gravatar from '../../common/Gravatar';
import PanelButton from '../../common/PanelButton';
import PrincipalAutocomplete, { Option, ServiceAccountOption, TeamOption, UserOption } from '../../groups/managedidentity/rules/PrincipalAutocomplete';
import PackageAutocomplete, { PackageOption } from '../../groups/package/PackageAutocomplete';
import PolicyFormScopeRule from './PolicyFormScopeRule';
import { blankScopeRule, isScopeRuleComplete, ScopeRuleFormData } from './scopeRules';

// The empty string is the unselected state: a policy's type has no sensible default, so it has to be
// chosen before there is anything else to fill in.
export type PolicyKind = '' | 'OPA';
export type PolicyStage = 'PRE_PLAN' | 'POST_PLAN';
export type PolicyEnforcementLevel = 'ADVISORY' | 'SOFT_MANDATORY' | 'HARD_MANDATORY';
// A run with no apply cannot be enforced at SOFT_MANDATORY: an override there would unblock nothing,
// and nobody is waiting on a speculative plan or a scheduled assessment run to approve one. The api
// rejects it, so the type leaves it out rather than letting the form offer it.
export type SpeculativeRunEnforcementLevel = 'ADVISORY' | 'HARD_MANDATORY';

const StyledAvatar = styled(Avatar)(() => ({
    width: 24,
    height: 24,
    marginRight: 2,
    backgroundColor: 'avatar.default',
}));

export interface PolicyFormData {
    kind: PolicyKind;
    name: string;
    description: string;
    package: PackageOption | null;
    packageVersionConstraint: string;
    packageDigest: string;
    stage: PolicyStage;
    enforcementLevel: PolicyEnforcementLevel;
    speculativeRunEnforcementLevel: SpeculativeRunEnforcementLevel;
    requiredApprovals: number;
    allowedUsers: any[];
    allowedTeams: any[];
    allowedServiceAccounts: any[];
    scope: ScopeRuleFormData[];
}

export const DEFAULT_POLICY_FORM_DATA: PolicyFormData = {
    kind: '',
    name: '',
    description: '',
    package: null,
    packageVersionConstraint: '',
    packageDigest: '',
    stage: 'POST_PLAN',
    enforcementLevel: 'ADVISORY',
    speculativeRunEnforcementLevel: 'ADVISORY',
    // Zero means a soft failure is overridable by anyone holding the permission, with no gate to wait
    // on. That is the lighter of the two behaviours, so it is where a new policy starts.
    requiredApprovals: 0,
    allowedUsers: [],
    allowedTeams: [],
    allowedServiceAccounts: [],
    scope: [],
};

// KIND_OPTIONS lists the supported policy types. OPA is the only one today; further types are added
// here alongside their own configuration section in the form.
const KIND_OPTIONS: { value: PolicyKind, label: string, description: string }[] = [
    { value: 'OPA', label: 'OPA', description: 'Open Policy Agent (Rego) policy evaluated against runs.' },
];

// STAGE_OPTIONS exposes the policy evaluation stages the backend supports.
const STAGE_OPTIONS: { value: PolicyStage, label: string }[] = [
    { value: 'PRE_PLAN', label: 'Pre Plan' },
    { value: 'POST_PLAN', label: 'Post Plan' },
];

// The stage label, keyed by the GraphQL enum value, so any other view that shows a policy's stage
// (the policy card, the detail page) matches this form's wording instead of showing the raw constant.
export const STAGE_LABELS: Record<string, string> = Object.fromEntries(
    STAGE_OPTIONS.map(o => [o.value, o.label])
);

const ENFORCEMENT_OPTIONS: { value: PolicyEnforcementLevel, label: string, description: string }[] = [
    { value: 'ADVISORY', label: 'Advisory', description: 'Failures are logged but never block the run.' },
    { value: 'SOFT_MANDATORY', label: 'Soft Mandatory', description: 'Failures block the run but can be overridden.' },
    { value: 'HARD_MANDATORY', label: 'Hard Mandatory', description: 'Failures always block the run and cannot be overridden.' },
];

// Mirrors STAGE_LABELS, for the enforcement level. Also used for the speculative level, whose values
// are a subset.
export const ENFORCEMENT_LEVEL_LABELS: Record<string, string> = Object.fromEntries(
    ENFORCEMENT_OPTIONS.map(o => [o.value, o.label])
);

// A speculative plan and an assessment run are both plan-only, so the two options are the only things
// a failure can do to one: be recorded, or stop it happening at all.
const SPECULATIVE_ENFORCEMENT_OPTIONS: { value: SpeculativeRunEnforcementLevel, label: string, description: string }[] = [
    { value: 'ADVISORY', label: 'Advisory', description: 'Failures are logged but never block a speculative plan or an assessment run.' },
    { value: 'HARD_MANDATORY', label: 'Hard Mandatory', description: 'Failures always block a speculative plan or an assessment run, and cannot be overridden.' },
];

// The api requires a policy that asks for approvals to name at least one principal who can give them,
// so a count above zero with an empty approver list is a save that would fail. Callers use this to
// disable their submit button instead.
export function isMissingApprovers(data: PolicyFormData): boolean {
    return data.enforcementLevel === 'SOFT_MANDATORY'
        && data.requiredApprovals > 0
        && data.allowedUsers.length + data.allowedTeams.length + data.allowedServiceAccounts.length === 0;
}

export function buildApproverInput(data: PolicyFormData) {
    return data.enforcementLevel === 'SOFT_MANDATORY' && data.requiredApprovals > 0
        ? {
            requiredApprovals: data.requiredApprovals,
            allowedUsers: data.allowedUsers.map((u: any) => u.id),
            allowedServiceAccounts: data.allowedServiceAccounts.map((s: any) => s.id),
            allowedTeams: data.allowedTeams.map((t: any) => t.id),
        }
        : {};
}

interface Props {
    groupPath: string;
    data: PolicyFormData;
    onChange: (data: PolicyFormData) => void;
    error?: MutationError;
    editMode?: boolean;
}

function PolicyForm({ groupPath, data, onChange, error, editMode }: Props) {
    const theme = useTheme();

    // A rule row is always on screen so there is nothing to click before typing in one. While the policy
    // has no rules that row is virtual rather than seeded into form state, so a form the user never
    // touches still submits an empty scope.
    const blankRow = useRef(blankScopeRule()).current;
    const scopeRows = data.scope.length > 0 ? data.scope : [blankRow];

    // PanelButton's disabled prop only styles it, so edit mode has to be enforced here as well —
    // the same guard ManagedIdentityForm applies to its type panels.
    const onKindChange = (kind: PolicyKind) => {
        if (!editMode) {
            onChange({ ...data, kind });
        }
    };

    const onScopeRuleChange = (rule: ScopeRuleFormData) => {
        onChange({ ...data, scope: scopeRows.map(r => r._id === rule._id ? rule : r) });
    };

    // Removing the last row leaves an empty scope, which puts the virtual blank row back on screen.
    const onDeleteScopeRule = (id: string) => {
        onChange({ ...data, scope: scopeRows.filter(r => r._id !== id) });
    };

    const onNewScopeRule = () => {
        onChange({ ...data, scope: [...scopeRows, blankScopeRule()] });
    };

    const handleDeletePrincipal = (principal: any) => {
        function deleteFrom(field: keyof PolicyFormData) {
            const copy = [...(data[field] as any[])];
            const index = copy.findIndex(item => item.id === principal.id);
            if (index !== -1) {
                copy.splice(index, 1);
                onChange({ ...data, [field]: copy });
            }
        }
        switch (principal.type) {
            case 'user': deleteFrom('allowedUsers'); break;
            case 'team': deleteFrom('allowedTeams'); break;
            case 'serviceaccount': deleteFrom('allowedServiceAccounts'); break;
        }
    };

    const onPrincipalSelected = (value: Option | null) => {
        if (!value) return;
        switch (value.type) {
            case 'user': {
                const user = value as UserOption;
                onChange({ ...data, allowedUsers: [...data.allowedUsers, { id: user.id, email: user.email, username: user.username }] });
                break;
            }
            case 'team': {
                const team = value as TeamOption;
                onChange({ ...data, allowedTeams: [...data.allowedTeams, { id: team.id, name: team.name }] });
                break;
            }
            case 'serviceaccount': {
                const sa = value as ServiceAccountOption;
                onChange({ ...data, allowedServiceAccounts: [...data.allowedServiceAccounts, { id: sa.id, name: sa.name, resourcePath: sa.resourcePath }] });
                break;
            }
        }
    };

    const principals = [
        ...data.allowedUsers.map((user: any) => ({ id: user.id, type: 'user', label: user.email, tooltip: user.email, name: user.username })),
        ...data.allowedTeams.map((team: any) => ({ id: team.id, type: 'team', label: team.name[0].toUpperCase(), tooltip: team.name, name: team.name })),
        ...data.allowedServiceAccounts.map((sa: any) => ({ id: sa.id, type: 'serviceaccount', label: sa.name[0].toUpperCase(), tooltip: sa.resourcePath, name: sa.resourcePath }))
    ];

    const selectedIds = principals.reduce((acc: Set<string>, item: any) => {
        acc.add(item.id);
        return acc;
    }, new Set<string>());

    const hasIncompleteScopeRule = data.scope.some(rule => !isScopeRuleComplete(rule));

    return (
        <Box>
            {error && <Alert sx={{ marginBottom: 2 }} severity={error.severity}>
                {error.message}
            </Alert>}
            {/* The type comes first because it decides which configuration section applies. It is
                fixed once the policy exists: a policy's data is stored per kind. */}
            <Box sx={{ mb: 4 }}>
                <Typography variant="subtitle1" gutterBottom>Select Type</Typography>
                <Divider light />
                <Stack marginTop={2} direction={{ xs: 'column', md: 'row' }} spacing={2} useFlexGap flexWrap="wrap">
                    {KIND_OPTIONS.map(opt => <PanelButton
                        key={opt.value}
                        disabled={editMode}
                        selected={data.kind === opt.value}
                        onClick={() => onKindChange(opt.value)}
                    >
                        <Typography variant="subtitle1">{opt.label}</Typography>
                        <Typography variant="caption" align="center">
                            {opt.description}
                        </Typography>
                    </PanelButton>)}
                </Stack>
            </Box>
            <Typography variant="subtitle1" gutterBottom>Details</Typography>
            <Divider light />
            <Box sx={{ mt: 2, mb: 4 }}>
                <Box sx={{ mb: 2 }}>
                    <TextField
                        size="small"
                        fullWidth
                        required
                        label="Name"
                        placeholder="e.g. compliance-policy"
                        helperText="A unique name for this policy within the group."
                        value={data.name}
                        disabled={editMode}
                        onChange={event => onChange({ ...data, name: event.target.value })}
                    />
                </Box>
                <TextField
                    size="small"
                    fullWidth
                    label="Description (optional)"
                    placeholder="What does this policy enforce?"
                    value={data.description}
                    onChange={event => onChange({ ...data, description: event.target.value })}
                />
            </Box>
            {/* Everything from here to the enforcement level is OPA's own configuration — it is what
                gets sent as opaData — so it stays hidden until OPA is the chosen type. */}
            {data.kind === 'OPA' && <>
                <Typography variant="subtitle1" gutterBottom>OPA Configuration</Typography>
                <Divider light />
                <Box sx={{ mt: 2, mb: 4 }}>
                    <Typography variant="subtitle2" gutterBottom>Package</Typography>
                    {/* The field stays on screen pre-filled in both modes rather than collapsing to a chip
                once a package is chosen: it accepts free-form input, which is captured as it is typed,
                so collapsing would happen on the first keystroke. */}
                    <Box sx={{ mb: 2 }}>
                        <PackageAutocomplete
                            groupPath={groupPath}
                            value={data.package}
                            onSelected={(value) => onChange({ ...data, package: value })}
                            filterOptions={(options) => options}
                        />
                        {/* The field itself already shows the fully-qualified source, so this is only a hint
                    for an empty field — including that an unpublished package may be named by hand. */}
                        {!data.package?.packageSource && <Typography variant="caption" color="textSecondary" display="block" sx={{ mt: 0.5 }}>
                            Pick a package, or type the source of one that is not published yet, e.g. my-group/sub-group/my-package.
                        </Typography>}
                    </Box>
                    <Box sx={{ mb: 2 }}>
                        <TextField
                            size="small"
                            fullWidth
                            label="Version constraint"
                            placeholder="e.g. 1.0.0 or ^1.0"
                            helperText="Leave blank to always use the latest version."
                            value={data.packageVersionConstraint}
                            onChange={event => onChange({ ...data, packageVersionConstraint: event.target.value })}
                        />
                    </Box>
                    <Box sx={{ mb: 2 }}>
                        <TextField
                            size="small"
                            fullWidth
                            label="Digest (optional)"
                            placeholder="hex sha256"
                            helperText="Paste a version's checksum to fail the run if the policy content changes."
                            value={data.packageDigest}
                            onChange={event => onChange({ ...data, packageDigest: event.target.value })}
                        />
                    </Box>
                    <Box sx={{ mb: 2 }}>
                        <FormControl size="small" sx={{ minWidth: 160 }}>
                            <InputLabel>Stage</InputLabel>
                            <Select
                                label="Stage"
                                value={data.stage}
                                onChange={event => onChange({ ...data, stage: event.target.value as PolicyStage })}
                            >
                                {STAGE_OPTIONS.map(opt => (
                                    <MenuItem key={opt.value} value={opt.value}>
                                        {opt.label}
                                    </MenuItem>
                                ))}
                            </Select>
                        </FormControl>
                    </Box>
                    <Box>
                        <Typography variant="subtitle2" mb={2}>Enforcement Level</Typography>
                        <Box sx={{ display: 'flex', gap: 3, flexWrap: 'wrap', mb: 1 }}>
                            <FormControl size="small" sx={{ minWidth: 420 }}>
                                <InputLabel>Apply Runs</InputLabel>
                                <Select
                                    label="Apply Runs"
                                    value={data.enforcementLevel}
                                    onChange={event => {
                                        const level = event.target.value as PolicyEnforcementLevel;
                                        onChange({ ...data, enforcementLevel: level });
                                    }}
                                >
                                    {ENFORCEMENT_OPTIONS.map(opt => (
                                        <MenuItem key={opt.value} value={opt.value}>
                                            <ListItemText
                                                primary={opt.label}
                                                secondary={ENFORCEMENT_OPTIONS.find(o => o.value === opt.value)?.description}
                                            />
                                        </MenuItem>
                                    ))}
                                </Select>
                            </FormControl>
                            <FormControl size="small" sx={{ minWidth: 420 }}>
                                <InputLabel>Speculative & Assessment Runs</InputLabel>
                                <Select
                                    label="Speculative & Assessment Runs"
                                    value={data.speculativeRunEnforcementLevel}
                                    onChange={event => onChange({ ...data, speculativeRunEnforcementLevel: event.target.value as SpeculativeRunEnforcementLevel })}
                                >
                                    {SPECULATIVE_ENFORCEMENT_OPTIONS.map(opt => (
                                        <MenuItem key={opt.value} value={opt.value}>
                                            <ListItemText
                                                primary={opt.label}
                                                secondary={ENFORCEMENT_OPTIONS.find(o => o.value === opt.value)?.description}
                                            />
                                        </MenuItem>
                                    ))}
                                </Select>
                            </FormControl>
                        </Box>
                    </Box>
                </Box>
            </>}

            <Typography variant="subtitle1" gutterBottom>Scope Rules</Typography>
            <Divider light />
            <Box sx={{ mt: 2, mb: 4 }}>
                <Typography variant="caption" color="textSecondary" display="block" sx={{ mb: 1.5 }}>
                    Restrict which workspaces this policy evaluates on. Empty scope applies to all workspaces under the owning group.
                    Excludes are checked first; if an exclude matches, the policy is skipped regardless of includes. Scope rules support
                    path glob patterns or TRNs.
                </Typography>
                <Stack spacing={2} sx={{ mb: 1.5 }}>
                    {scopeRows.map(rule => (
                        <PolicyFormScopeRule
                            key={rule._id}
                            rule={rule}
                            onChange={onScopeRuleChange}
                            onDelete={() => onDeleteScopeRule(rule._id)}
                            disableDelete={scopeRows.length === 1 && !rule.pattern}
                        />
                    ))}
                </Stack>
                {hasIncompleteScopeRule && (
                    <Typography variant="caption" color="textSecondary" display="block" sx={{ mb: 1.5 }}>
                        Rules without a value are ignored.
                    </Typography>
                )}
                <Button
                    size="small"
                    variant="outlined"
                    startIcon={<AddIcon />}
                    onClick={onNewScopeRule}
                >
                    Add Rule
                </Button>
            </Box>

            {/* Approvals are a policy-level setting rather than part of any type's configuration, so
                they are not tied to the selected type — only to the level that can be overridden. */}
            {data.enforcementLevel === 'SOFT_MANDATORY' && <Box>
                <Typography variant="subtitle1" gutterBottom>Approvals</Typography>
                <Divider light />
                <Typography sx={{ mt: 2, mb: 2 }} color="textSecondary" variant="body2">
                    Approvals are optional. At zero, a soft failure can be overridden by anyone with
                    permission to do so. Above zero, the run waits for that many approvals from the
                    approvers named below.
                </Typography>
                <Box marginBottom={3}>
                    <Typography sx={{ mb: 1 }} variant="body1">Required Approvals</Typography>
                    <TextField
                        type="number"
                        size="small"
                        sx={{ width: 120 }}
                        slotProps={{ htmlInput: { min: 0 } }}
                        value={data.requiredApprovals}
                        onChange={(e) => onChange({ ...data, requiredApprovals: Math.max(0, parseInt(e.target.value, 10) || 0) })}
                    />
                </Box>
                {/* Approvers only mean something once an approval is required, and the api rejects a
                    policy that requires one without naming anyone who can give it — so the list appears
                    with the requirement, and is mandatory whenever it appears. The selection is left in
                    form state when the count returns to zero, so putting it back restores the list. */}
                {data.requiredApprovals > 0 && <>
                    <Typography sx={{ mb: 1 }} variant="body1">Allowed Approvers</Typography>
                    <Box sx={{ border: `1px solid ${theme.palette.divider}`, borderRadius: '4px' }} marginBottom={2} padding={2}>
                        <Box sx={{ marginBottom: 2 }}>
                            <PrincipalAutocomplete
                                groupPath={groupPath}
                                onSelected={onPrincipalSelected}
                                filterOptions={(options: Option[]) => options.filter(option => !selectedIds.has(option.id))}
                            />
                        </Box>
                        <Typography color="textSecondary">
                            {principals.length} approver{principals.length === 1 ? '' : 's'} selected
                        </Typography>
                        <List dense>
                            {principals.map((pr: any) => (
                                <ListItem
                                    disableGutters
                                    secondaryAction={<IconButton onClick={() => handleDeletePrincipal(pr)}>
                                        <DeleteIcon />
                                    </IconButton>}
                                    key={pr.id}>
                                    {pr.type === 'user' && <Gravatar sx={{ marginRight: 1 }} width={24} height={24} email={pr.label} />}
                                    {pr.type !== 'user' && <StyledAvatar sx={{ marginRight: 1 }}>{pr.label}</StyledAvatar>}
                                    <ListItemText primary={pr.name} primaryTypographyProps={{ noWrap: true }} />
                                </ListItem>))}
                        </List>
                    </Box>
                    {isMissingApprovers(data) && <Typography variant="caption" color="warning" display="block">
                        Add at least one approver, or set required approvals back to zero.
                    </Typography>}
                </>}
            </Box>}
        </Box>
    );
}

export default PolicyForm;
