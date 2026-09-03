import AddIcon from '@mui/icons-material/Add';
import DeleteIcon from '@mui/icons-material/Delete';
import { Alert, Avatar, Box, Button, Checkbox, Divider, FormControlLabel, IconButton, List, ListItem, ListItemText, Stack, styled, TextField, Typography, useTheme } from '@mui/material';
import { useRef } from 'react';
import { MutationError } from '../../common/error';
import Gravatar from '../../common/Gravatar';
import PanelButton from '../../common/PanelButton';
import PrincipalAutocomplete, { Option, ServiceAccountOption, TeamOption, UserOption } from '../../groups/managedidentity/rules/PrincipalAutocomplete';
import PackageAutocomplete, { PackageOption } from '../../groups/package/PackageAutocomplete';
import EnforcementLevelSelects from './EnforcementLevelSelects';
import { ENFORCEMENT_LEVEL_LABELS, KIND_LABELS, STAGE_LABELS } from './policyDisplay';
import PolicyFormScopeRule from './PolicyFormScopeRule';
import { blankScopeRule, isScopeRuleComplete, ScopeRuleFormData } from './scopeRules';
import StageSelector from './StageSelector';

// The empty string is the unselected state: a policy's type has no sensible default, so it has to be
// chosen before there is anything else to fill in.
export type PolicyKind = '' | 'OPA' | 'MODULE_ATTESTATION';
// Likewise, the stage has no sensible default — evaluating too early or too late has real
// consequences — so it also starts unselected rather than silently picking one for the user.
export type PolicyStage = '' | 'PRE_PLAN' | 'POST_PLAN' | 'PRE_APPLY' | 'POST_APPLY';
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

// OPAPolicyFormData is OPA's own configuration, parallel to ModuleAttestationPolicyFormData below.
export interface OPAPolicyFormData {
    package: PackageOption | null;
    packageVersionConstraint: string;
    packageDigest: string;
}

// ModuleAttestationPolicyFormData is module attestation's own configuration, parallel to
// OPAPolicyFormData above.
export interface ModuleAttestationPolicyFormData {
    publicKey: string;
    predicateType: string;
    verifyStateLineage: boolean;
}

export interface PolicyFormData {
    kind: PolicyKind;
    name: string;
    description: string;
    opa: OPAPolicyFormData;
    moduleAttestation: ModuleAttestationPolicyFormData;
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
    opa: {
        package: null,
        packageVersionConstraint: '',
        packageDigest: '',
    },
    moduleAttestation: {
        publicKey: '',
        predicateType: '',
        verifyStateLineage: false,
    },
    stage: '',
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

// KIND_OPTIONS lists the supported policy types; further types are added here alongside their own
// configuration section in the form. Labels come from KIND_LABELS (see policyDisplay) so the option
// text always matches what every other view calls the same kind.
const KIND_OPTIONS: { value: PolicyKind, label: string, description: string }[] = [
    { value: 'OPA', label: KIND_LABELS.OPA, description: 'Open Policy Agent (Rego) policy evaluated against runs.' },
    {
        value: 'MODULE_ATTESTATION',
        label: KIND_LABELS.MODULE_ATTESTATION,
        description: 'Requires the module a run deploys to carry an in-toto attestation signed by a specified key.',
    },
];

// ALL_STAGE_OPTIONS exposes every policy evaluation stage the backend supports. Labels come from
// STAGE_LABELS (see policyDisplay) so the option text always matches what every other view calls the
// same stage.
const ALL_STAGE_OPTIONS: { value: PolicyStage, label: string, description: string }[] = [
    { value: 'PRE_PLAN', label: STAGE_LABELS.PRE_PLAN, description: 'Evaluated before the plan is generated.' },
    { value: 'POST_PLAN', label: STAGE_LABELS.POST_PLAN, description: 'Evaluated against the generated plan, before apply.' },
    { value: 'PRE_APPLY', label: STAGE_LABELS.PRE_APPLY, description: 'Evaluated immediately before apply runs.' },
    { value: 'POST_APPLY', label: STAGE_LABELS.POST_APPLY, description: 'Evaluated after apply, once state has been written.' },
];

// stageOptionsForKind restricts the offered stages to the ones the chosen kind's data can actually
// evaluate at. A module is verified before it is used, so a post-plan or post-apply check would come
// too late to stop anything — the api rejects those stages for this kind (see
// models.ModuleAttestationStages on the backend).
export function stageOptionsForKind(kind: PolicyKind): { value: PolicyStage, label: string, description: string }[] {
    if (kind === 'MODULE_ATTESTATION') {
        return ALL_STAGE_OPTIONS.filter(opt => opt.value === 'PRE_PLAN' || opt.value === 'PRE_APPLY');
    }
    return ALL_STAGE_OPTIONS;
}

// STAGE_COLUMN_COUNT is every kind's stage grid sized the same, to the full stage list, rather than
// each kind's own (possibly narrower) option count — so a kind offering fewer stages doesn't get
// wider columns than one offering all of them, and the grid does not reflow if a kind's option list
// changes.
export const STAGE_COLUMN_COUNT = ALL_STAGE_OPTIONS.length;

// isPostApplyStage reports whether the given stage is restricted to advisory enforcement: state has
// already been written by the time a post-apply check evaluates, so there is no run outcome left for
// a stronger enforcement level to protect (see models.Policy.Validate on the backend, which rejects
// anything else for this stage).
export function isPostApplyStage(stage: PolicyStage): boolean {
    return stage === 'POST_APPLY';
}

// ENFORCEMENT_OPTIONS lists the enforcement levels a policy can be created or edited with. Labels
// come from ENFORCEMENT_LEVEL_LABELS (see policyDisplay) so the option text always matches what
// every other view calls the same level.
export const ENFORCEMENT_OPTIONS: { value: PolicyEnforcementLevel, label: string, description: string }[] = [
    { value: 'ADVISORY', label: ENFORCEMENT_LEVEL_LABELS.ADVISORY, description: 'Failures are logged but never block the run.' },
    { value: 'SOFT_MANDATORY', label: ENFORCEMENT_LEVEL_LABELS.SOFT_MANDATORY, description: 'Failures block the run but can be overridden.' },
    { value: 'HARD_MANDATORY', label: ENFORCEMENT_LEVEL_LABELS.HARD_MANDATORY, description: 'Failures always block the run and cannot be overridden.' },
];

// A speculative plan and an assessment run are both plan-only, so the two options are the only things
// a failure can do to one: be recorded, or stop it happening at all. Labels come from
// ENFORCEMENT_LEVEL_LABELS (see policyDisplay) so the option text always matches what every other
// view calls the same level.
export const SPECULATIVE_ENFORCEMENT_OPTIONS: { value: SpeculativeRunEnforcementLevel, label: string, description: string }[] = [
    { value: 'ADVISORY', label: ENFORCEMENT_LEVEL_LABELS.ADVISORY, description: 'Failures are logged but never block a speculative plan or an assessment run.' },
    { value: 'HARD_MANDATORY', label: ENFORCEMENT_LEVEL_LABELS.HARD_MANDATORY, description: 'Failures always block a speculative plan or an assessment run, and cannot be overridden.' },
];

// The api requires a policy that asks for approvals to name at least one principal who can give them,
// so a count above zero with an empty approver list is a save that would fail. Callers use this to
// disable their submit button instead.
export function isMissingApprovers(data: PolicyFormData): boolean {
    return data.enforcementLevel === 'SOFT_MANDATORY'
        && data.requiredApprovals > 0
        && data.allowedUsers.length + data.allowedTeams.length + data.allowedServiceAccounts.length === 0;
}

// hasRequiredKindData reports whether the fields the chosen kind's data can't be submitted without
// have been filled in: the stage (there is no sensible default for it), plus a package for OPA or a
// public key for module attestation. Callers use this to disable their submit button until it is.
export function hasRequiredKindData(data: PolicyFormData): boolean {
    if (!data.stage) {
        return false;
    }
    return data.kind === 'MODULE_ATTESTATION'
        ? !!data.moduleAttestation.publicKey.trim()
        : !!data.opa.package?.packageSource;
}

// buildKindDataInput builds the opaData or moduleAttestationData half of the create/update mutation
// input from the form, whichever the chosen kind uses. Called after hasRequiredKindData confirms the
// kind's required field is present, which is why the stage cast below is safe.
export function buildKindDataInput(data: PolicyFormData) {
    const stage = data.stage as Exclude<PolicyStage, ''>;
    if (data.kind === 'MODULE_ATTESTATION') {
        return {
            moduleAttestationData: {
                publicKey: data.moduleAttestation.publicKey.trim(),
                predicateType: data.moduleAttestation.predicateType.trim() || null,
                verifyStateLineage: data.moduleAttestation.verifyStateLineage,
                stage,
                enforcementLevel: data.enforcementLevel,
                speculativeRunEnforcementLevel: data.speculativeRunEnforcementLevel,
            },
        };
    }
    return {
        opaData: {
            packageSource: data.opa.package!.packageSource,
            packageVersionConstraint: data.opa.packageVersionConstraint.trim() || null,
            packageDigest: data.opa.packageDigest.trim() || null,
            stage,
            enforcementLevel: data.enforcementLevel,
            speculativeRunEnforcementLevel: data.speculativeRunEnforcementLevel,
        },
    };
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

// clearApprovals drops the approval configuration — the required count and every named approver.
// Approvals gate the override of an apply blocked at soft mandatory, so they only mean anything at
// that level for apply runs; changing that level therefore clears them rather than leaving them
// hidden in form state, where they would silently reappear if the user returned to soft mandatory.
function clearApprovals(data: PolicyFormData): PolicyFormData {
    return {
        ...data,
        requiredApprovals: DEFAULT_POLICY_FORM_DATA.requiredApprovals,
        allowedUsers: [],
        allowedTeams: [],
        allowedServiceAccounts: [],
    };
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
    // the same guard ManagedIdentityForm applies to its type panels. Switching kind can leave the
    // stage on a value the new kind doesn't support (e.g. module attestation dropped from
    // post_apply), so it is reset back to unselected rather than left stale, or silently defaulted
    // to some other stage, on screen.
    const onKindChange = (kind: PolicyKind) => {
        if (editMode) {
            return;
        }
        const validStages = stageOptionsForKind(kind);
        const stage = validStages.some(opt => opt.value === data.stage) ? data.stage : '';
        onChange({ ...data, kind, stage });
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
                    <Box sx={{ mb: 3 }}>
                        <Typography variant="subtitle2" gutterBottom>Stage</Typography>
                        <StageSelector
                            stage={data.stage}
                            options={stageOptionsForKind(data.kind)}
                            columnCount={STAGE_COLUMN_COUNT}
                            onSelect={stage => {
                                // Post-apply can only be advisory (state has already been written by
                                // the time it evaluates), so switching to it pins both enforcement
                                // levels rather than leaving a stale, now-invalid selection on screen.
                                // Pinning apply runs to advisory takes the approvals with it, the same
                                // as choosing that level directly.
                                onChange(isPostApplyStage(stage)
                                    ? clearApprovals({ ...data, stage, enforcementLevel: 'ADVISORY', speculativeRunEnforcementLevel: 'ADVISORY' })
                                    : { ...data, stage });
                            }}
                        />
                    </Box>
                    <Box sx={{ mb: 3 }}>
                        <Typography variant="subtitle2" mb={2}>Enforcement Level</Typography>
                        <EnforcementLevelSelects
                            enforcementLevel={data.enforcementLevel}
                            speculativeRunEnforcementLevel={data.speculativeRunEnforcementLevel}
                            disabled={isPostApplyStage(data.stage)}
                            onEnforcementLevelChange={level => {
                                // The approvals below belong to the level being left behind, so
                                // they go with it — including when the new level is soft
                                // mandatory, where the section reappears configured fresh.
                                onChange(clearApprovals({ ...data, enforcementLevel: level }));
                            }}
                            onSpeculativeRunEnforcementLevelChange={level => onChange({ ...data, speculativeRunEnforcementLevel: level })}
                        />
                    </Box>
                    <Typography variant="subtitle2" gutterBottom>Package</Typography>
                    {/* The field stays on screen pre-filled in both modes rather than collapsing to a chip
                once a package is chosen: it accepts free-form input, which is captured as it is typed,
                so collapsing would happen on the first keystroke. */}
                    <Box sx={{ mb: 2 }}>
                        <PackageAutocomplete
                            groupPath={groupPath}
                            value={data.opa.package}
                            onSelected={(value) => onChange({ ...data, opa: { ...data.opa, package: value } })}
                            filterOptions={(options) => options}
                        />
                        {/* The field itself already shows the fully-qualified source, so this is only a hint
                    for an empty field — including that an unpublished package may be named by hand. */}
                        {!data.opa.package?.packageSource && <Typography variant="caption" color="textSecondary" display="block" sx={{ mt: 0.5 }}>
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
                            value={data.opa.packageVersionConstraint}
                            onChange={event => onChange({ ...data, opa: { ...data.opa, packageVersionConstraint: event.target.value } })}
                        />
                    </Box>
                    <Box sx={{ mb: 2 }}>
                        <TextField
                            size="small"
                            fullWidth
                            label="Digest (optional)"
                            placeholder="hex sha256"
                            helperText="Paste a version's checksum to fail the run if the policy content changes."
                            value={data.opa.packageDigest}
                            onChange={event => onChange({ ...data, opa: { ...data.opa, packageDigest: event.target.value } })}
                        />
                    </Box>
                </Box>
            </>}

            {/* Module attestation's own configuration, parallel to the OPA section above — what gets
                sent as moduleAttestationData. Its stage options exclude post-plan/post-apply (see
                stageOptionsForKind), so there is no post-apply-advisory clamp to apply here. */}
            {data.kind === 'MODULE_ATTESTATION' && <>
                <Typography variant="subtitle1" gutterBottom>Module Attestation Configuration</Typography>
                <Divider light />
                <Box sx={{ mt: 2, mb: 4 }}>
                    <Box sx={{ mb: 2 }}>
                        <Typography variant="subtitle2" gutterBottom>Stage</Typography>
                        <StageSelector
                            stage={data.stage}
                            options={stageOptionsForKind(data.kind)}
                            columnCount={STAGE_COLUMN_COUNT}
                            onSelect={stage => onChange({ ...data, stage })}
                        />
                    </Box>
                    <Box sx={{ mb: 2 }}>
                        <Typography variant="subtitle2" mb={2}>Enforcement Level</Typography>
                        <EnforcementLevelSelects
                            enforcementLevel={data.enforcementLevel}
                            speculativeRunEnforcementLevel={data.speculativeRunEnforcementLevel}
                            onEnforcementLevelChange={level => onChange(clearApprovals({ ...data, enforcementLevel: level }))}
                            onSpeculativeRunEnforcementLevelChange={level => onChange({ ...data, speculativeRunEnforcementLevel: level })}
                        />
                    </Box>
                    <Box sx={{ mb: 2 }}>
                        <Typography variant="subtitle2" gutterBottom>Public Key</Typography>
                        <TextField
                            size="small"
                            fullWidth
                            required
                            multiline
                            rows={6}
                            placeholder="-----BEGIN PUBLIC KEY-----"
                            helperText="The PEM-encoded public key the module's attestation signature must verify against. Supports ECDSA and RSA public keys."
                            value={data.moduleAttestation.publicKey}
                            onChange={event => onChange({ ...data, moduleAttestation: { ...data.moduleAttestation, publicKey: event.target.value } })}
                        />
                    </Box>
                    <Box sx={{ mb: 2 }}>
                        <TextField
                            size="small"
                            fullWidth
                            label="In-Toto Predicate type (optional)"
                            placeholder="e.g. https://slsa.dev/provenance/v1"
                            helperText="Leave blank to accept any predicate type."
                            value={data.moduleAttestation.predicateType}
                            onChange={event => onChange({ ...data, moduleAttestation: { ...data.moduleAttestation, predicateType: event.target.value } })}
                        />
                    </Box>
                    <Box sx={{ mb: 2 }}>
                        <FormControlLabel
                            control={
                                <Checkbox
                                    checked={data.moduleAttestation.verifyStateLineage}
                                    onChange={event => onChange({ ...data, moduleAttestation: { ...data.moduleAttestation, verifyStateLineage: event.target.checked } })}
                                />
                            }
                            label="Verify state lineage"
                        />
                        <Typography variant="caption" color="textSecondary" display="block">
                            Also require the workspace's current state to have been written by a run using the same module source,
                            so state cannot be carried over from an unattested module.
                        </Typography>
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
