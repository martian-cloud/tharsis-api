/**
 * rules.tsx — the cleanup policy kind registry.
 *
 * To add a new kind: add its rule interface and union entry in types.ts, then a CLEANUP_POLICY_KINDS
 * entry below. Everything else (tabs, JSON editor, mutation inputs) derives from the registry.
 */
import React from 'react';
import AddIcon from '@mui/icons-material/Add';
import CheckIcon from '@mui/icons-material/Check';
import {
    Autocomplete, Box, Chip, Stack, TextField, Typography,
} from '@mui/material';
import CleanupRuleList, { CleanupRuleTemplate } from './CleanupRuleList';
import CleanupRuleDialog from './CleanupRuleDialog';
import Verdict from './Verdict';
import {
    CleanupKindPanelProps, CleanupPolicyKind, CleanupRule, CleanupStrategy, Condition, PolicyDataKey,
    RunCleanupRule, TerraformModuleCleanupRule, TerraformProviderCleanupRule,
} from './types';
import ConditionText from './ConditionText';

// A new rule starts without a strategy; the dialog blocks saving until one is picked.
const NO_STRATEGY = '' as CleanupStrategy;

// Only finished runs are candidates for deletion.
const FINAL_RUN_STATUSES = ['planned_and_finished', 'errored', 'canceled', 'discarded'];

// ─── Shared UI helpers ────────────────────────────────────────────────────────

function GlobField<R extends Record<string, any>>({
    label,
    field,
    placeholder,
    rule,
    onChange,
}: {
    label: string;
    field: keyof R;
    placeholder: string;
    rule: R;
    onChange: (r: R) => void;
}) {
    const value = rule[field] as string;
    return (
        <TextField size="small" fullWidth label={label} placeholder={placeholder}
            value={value}
            helperText={value ? undefined : `Exact match or glob, e.g. ${placeholder}`}
            inputProps={{ maxLength: 100 }}
            onChange={e => onChange({ ...rule, [field]: e.target.value })} />
    );
}

// FilterChip is a pill-style toggle (a plus icon when off, a check when on) used for the run-kind
// filters, where each field is a tri-state boolean (true / null) rather than a plain checkbox.
function FilterChip({
    label,
    selected,
    onClick,
}: {
    label: string;
    selected: boolean;
    onClick: () => void;
}) {
    return (
        <Chip
            clickable
            variant="outlined"
            size="small"
            icon={selected ? <CheckIcon fontSize="small" /> : <AddIcon fontSize="small" />}
            label={label}
            onClick={onClick}
            sx={{
                borderColor: selected ? 'primary.main' : 'divider',
                color: selected ? 'primary.main' : 'text.primary',
                '& .MuiChip-icon': { color: selected ? 'primary.main' : 'text.secondary' },
            }}
        />
    );
}

function RunFilterFields({
    rule,
    onChange,
}: {
    rule: RunCleanupRule;
    onChange: (r: RunCleanupRule) => void;
}) {
    const toggle = (label: string, field: keyof RunCleanupRule) => (
        <FilterChip
            label={label}
            selected={rule[field] === true}
            onClick={() => onChange({ ...rule, [field]: rule[field] !== null ? null : true })}
        />
    );
    return (
        <Stack spacing={2}>
            <Box>
                <Typography variant="body2" sx={{ mb: 1 }}>Run kind</Typography>
                <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
                    {toggle('Speculative', 'speculative')}{toggle('Assessment', 'assessment')}
                </Stack>
                <Typography variant="caption" color="textSecondary" sx={{ display: 'block', mt: 0.5 }}>
                    Select none to match every run kind.
                </Typography>
            </Box>
            <Box>
                <Typography variant="body2" sx={{ mb: 1 }}>Statuses</Typography>
                <Autocomplete
                    multiple
                    size="small"
                    options={FINAL_RUN_STATUSES}
                    value={rule.status}
                    onChange={(_, value) => onChange({ ...rule, status: value })}
                    renderInput={params => <TextField {...params} placeholder={rule.status.length === 0 ? 'Add status' : undefined} />}
                />
                <Typography variant="caption" color="textSecondary" sx={{ display: 'block', mt: 0.5 }}>
                    Select none to match every finished run.
                </Typography>
            </Box>
        </Stack>
    );
}
// ─── Kind registry ────────────────────────────────────────────────────────────

interface KindDefinition<R extends CleanupRule> {
    label: string;
    description: string;
    safetyNote: React.ReactNode;
    resource: string;
    groupOnly: boolean;
    policyDataKey: PolicyDataKey;
    jsonSchemaProperties: Record<string, object>;
    availableStrategies?: CleanupStrategy[];
    newRule: () => R;
    templates: CleanupRuleTemplate<R>[];
    conditions: (rule: R) => Condition[];
    filterFields: (rule: R, onChange: (r: R) => void) => React.ReactNode;
    renderList: (props: CleanupKindPanelProps<R>) => React.ReactNode;
}

type AnyKindDefinition = KindDefinition<any>;

function makeKindDefinition<R extends CleanupRule>(def: Omit<KindDefinition<R>, 'renderList'>): KindDefinition<R> {
    const getSummary = (rule: R): React.ReactNode => {
        const conds = def.conditions(rule);
        return (
            <>
                This rule: <Verdict rule={rule} resource={def.resource} />
                {conds.length > 0 && <>
                    {' where '}
                    {conds.map((c, i) => (
                        <React.Fragment key={c.label}>
                            {i > 0 && ' and '}
                            <ConditionText label={c.label} value={c.value} />
                        </React.Fragment>
                    ))}
                </>}
            </>
        );
    };

    const renderList = (props: CleanupKindPanelProps<R>) => (
        <CleanupRuleList {...props}
            resource={def.resource} ariaLabel={`${def.resource} cleanup rules`}
            getConditions={def.conditions as (rule: CleanupRule) => Condition[]}
            renderDialog={({ rule, position, isNew, open, onClose, onSave }) => (
                <CleanupRuleDialog rule={rule} position={position} isNew={isNew} open={open}
                    resource={def.resource} onClose={onClose} onSave={onSave} templates={def.templates as any}
                    safetyNote={def.safetyNote} getSummary={getSummary}
                    availableStrategies={def.availableStrategies}>
                    {def.filterFields}
                </CleanupRuleDialog>
            )}
        />
    );
    return { ...def, renderList };
}

// ─── Rule builders ────────────────────────────────────────────────────────────

// Every rule carries all of its kind's fields, so a template only states what it changes and the
// builder fills in the rest, keeping a template's serialized form in agreement with the JSON editor
// (which omits fields still at their default).
const moduleRule = (overrides: Partial<TerraformModuleCleanupRule> = {}): TerraformModuleCleanupRule => ({
    strategy: NO_STRATEGY,
    description: '',
    deleteAfterDays: 0,
    nameGlob: '*',
    systemGlob: '*',
    versionGlob: '*',
    ...overrides,
});

const providerRule = (overrides: Partial<TerraformProviderCleanupRule> = {}): TerraformProviderCleanupRule => ({
    strategy: NO_STRATEGY,
    description: '',
    deleteAfterDays: 0,
    nameGlob: '*',
    versionGlob: '*',
    ...overrides,
});

const runRule = (overrides: Partial<RunCleanupRule> = {}): RunCleanupRule => ({
    strategy: NO_STRATEGY,
    description: '',
    keepMin: 0,
    deleteAfterDays: 0,
    speculative: null,
    assessment: null,
    status: [],
    ...overrides,
});

// ─── Registry templates ───────────────────────────────────────────────────────

// The module and provider registries take the same rules, so the two kinds share one set of
// templates; only the resource noun in the wording differs.
type VersionRuleOverrides = Partial<Omit<TerraformModuleCleanupRule, 'systemGlob' | 'keepMin'>>;

const VERSION_TEMPLATES: Array<{ label: string; detail: (resource: string) => string; overrides: VersionRuleOverrides }> = [
    {
        label: 'Protect a release line',
        detail: () => 'Never deletes anything matching 1.*',
        overrides: { strategy: 'PROTECT', versionGlob: '1.*', description: 'protect 1.x' },
    },
    {
        label: 'Protect a name pattern',
        detail: () => 'Never deletes anything matching aws-*',
        overrides: { strategy: 'PROTECT', nameGlob: 'aws-*', description: 'protect aws-*' },
    },
    {
        label: 'Expire old versions',
        detail: () => 'Deletes versions older than a year while always keeping the 5 newest',
        overrides: { strategy: 'AGE', deleteAfterDays: 365, description: 'expire old versions' },
    },
    {
        label: 'Expire old prereleases',
        detail: () => 'Deletes prerelease versions older than 90 days, keeping the 3 newest',
        overrides: { strategy: 'AGE', versionGlob: '*-*', deleteAfterDays: 90, description: 'expire old prereleases' },
    },
    {
        label: 'Expire old RCs',
        detail: () => 'Deletes release candidates older than 90 days, keeping the 3 newest',
        overrides: { strategy: 'AGE', versionGlob: '*-rc.*', deleteAfterDays: 90, description: 'expire old RCs' },
    },
];

function versionTemplates<R extends CleanupRule>(
    resource: string,
    build: (overrides: VersionRuleOverrides) => R,
): CleanupRuleTemplate<R>[] {
    return VERSION_TEMPLATES.map(({ label, detail, overrides }) => ({
        label,
        detail: detail(resource),
        rule: () => build(overrides),
    }));
}

const RUN_TEMPLATES: CleanupRuleTemplate<RunCleanupRule>[] = [
    {
        label: 'Trim speculative',
        detail: 'Keeps the 50 newest speculative plans and deletes older ones',
        rule: () => runRule({ strategy: 'COUNT', keepMin: 50, speculative: true, description: 'trim speculative' }),
    },
    {
        label: 'Trim assessments',
        detail: 'Keeps the 20 newest drift assessments and deletes older ones',
        rule: () => runRule({ strategy: 'COUNT', keepMin: 20, assessment: true, speculative: true, description: 'trim assessments' }),
    },
    {
        label: 'Trim unsuccessful runs',
        detail: 'Keeps the 10 most recent errored, canceled, or discarded runs and deletes the rest',
        rule: () => runRule({ strategy: 'COUNT', keepMin: 10, status: ['errored', 'canceled', 'discarded'], description: 'trim unsuccessful runs' }),
    },
    {
        label: 'Expire old speculative plans',
        detail: 'Deletes speculative plans older than 30 days, keeping the 10 newest',
        rule: () => runRule({ strategy: 'AGE', deleteAfterDays: 30, keepMin: 10, speculative: true, description: 'expire old speculative plans' }),
    },
    {
        label: 'Expire old runs',
        detail: 'Deletes finished runs older than 90 days that did not produce a state version, keeping the 20 newest',
        rule: () => runRule({ strategy: 'AGE', deleteAfterDays: 90, keepMin: 20, description: 'expire old runs' }),
    },
];

// ─── Kinds ────────────────────────────────────────────────────────────────────

const GLOB_SCHEMA = {
    nameGlob: { type: 'string', description: 'Matches the resource name' },
    versionGlob: { type: 'string', description: 'Matches the semantic version' },
};

// globConditions shows every glob field once any is constrained, rendering "*" fields as label-only prose so "anything" is never confused with a literal glob value.
function globConditions(rule: { nameGlob: string; versionGlob: string; systemGlob?: string }): Condition[] {
    const fields: Array<{ label: string; value: string }> = [
        { label: 'name matches', value: rule.nameGlob },
        ...(rule.systemGlob !== undefined ? [{ label: 'system matches', value: rule.systemGlob }] : []),
        { label: 'version matches', value: rule.versionGlob },
    ];

    return fields.filter(f => f.value !== '*');
}

export const CLEANUP_POLICY_KINDS: Record<CleanupPolicyKind, AnyKindDefinition> = {
    TERRAFORM_MODULES: makeKindDefinition<TerraformModuleCleanupRule>({
        label: 'Terraform Modules',
        description: 'Clean up Terraform module versions based on module names, systems, and version patterns.',
        safetyNote: <>The <strong>latest</strong> version of each module is never deleted.</>,
        resource: 'module version',
        groupOnly: true,
        policyDataKey: 'terraformModulePolicyData',
        availableStrategies: ['AGE', 'PROTECT'],
        jsonSchemaProperties: {
            ...GLOB_SCHEMA,
            systemGlob: { type: 'string', description: 'Matches the module system' },
        },
        newRule: moduleRule,
        templates: versionTemplates('module version', moduleRule),
        conditions: globConditions,
        filterFields: (r, setR) => (
            <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} alignItems="flex-start">
                <GlobField label="Module name" field="nameGlob" placeholder="aws-*" rule={r} onChange={setR} />
                <GlobField label="System" field="systemGlob" placeholder="aws" rule={r} onChange={setR} />
                <GlobField label="Module version" field="versionGlob" placeholder="1.*" rule={r} onChange={setR} />
            </Stack>
        ),
    }),

    TERRAFORM_PROVIDERS: makeKindDefinition<TerraformProviderCleanupRule>({
        label: 'Terraform Providers',
        description: 'Clean up Terraform provider versions based on provider names and version patterns.',
        safetyNote: <>The <strong>latest</strong> version of each provider is never deleted.</>,
        resource: 'provider version',
        groupOnly: true,
        policyDataKey: 'terraformProviderPolicyData',
        availableStrategies: ['AGE', 'PROTECT'],
        jsonSchemaProperties: GLOB_SCHEMA,
        newRule: providerRule,
        templates: versionTemplates('provider version', providerRule),
        conditions: globConditions,
        filterFields: (r, setR) => (
            <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} alignItems="flex-start">
                <GlobField label="Provider name" field="nameGlob" placeholder="aws-*" rule={r} onChange={setR} />
                <GlobField label="Provider version" field="versionGlob" placeholder="1.*" rule={r} onChange={setR} />
            </Stack>
        ),
    }),

    RUNS: makeKindDefinition<RunCleanupRule>({
        label: 'Runs',
        description: 'Clean up runs based on run types, statuses, and other run characteristics.',
        safetyNote: "Runs with a state version are never deleted and don't count toward the keep minimum.",
        resource: 'run',
        groupOnly: false,
        policyDataKey: 'runPolicyData',
        jsonSchemaProperties: {
            speculative: { type: ['boolean', 'null'] },
            assessment: { type: ['boolean', 'null'] },
            status: { type: 'array', items: { enum: FINAL_RUN_STATUSES } },
        },
        newRule: runRule,
        templates: RUN_TEMPLATES,
        filterFields: (r, setR) => <RunFilterFields rule={r} onChange={setR} />,
        conditions: (r) => {
            const conditions: Condition[] = [];
            if (r.speculative !== null) conditions.push({ label: r.speculative ? 'speculative' : 'no speculative' });
            if (r.assessment !== null) conditions.push({ label: r.assessment ? 'assessment' : 'no assessment' });
            if (r.status.length > 0) conditions.push({ label: 'status is any of', value: r.status.join(', ') });
            return conditions;
        },
    }),
};
