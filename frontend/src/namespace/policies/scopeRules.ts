import { type Theme } from '@mui/material/styles';
import { nanoid } from 'nanoid';

// These mirror the PolicyScopeRuleType and PolicyScopeRuleAction GraphQL enums. They are spelled out
// rather than typed as string so a value that does not match the wire casing cannot reach the
// mutation input.
export type ScopeRuleTypeValue = 'WORKSPACE' | 'GROUP' | 'MANAGED_IDENTITY';
export type ScopeRuleActionValue = 'INCLUDE' | 'EXCLUDE';

export interface ScopeRuleFormData {
    type: ScopeRuleTypeValue;
    action: ScopeRuleActionValue;
    pattern: string;
    /** Row key, stable for the lifetime of the row. Not sent to the backend. */
    _id: string;
}

interface ScopeTypeOption {
    value: ScopeRuleTypeValue;
    label: string;
    placeholder: string;
}

// Every type matches a path glob; they differ in which of the run's paths they are matched against. The
// group placeholder is deliberately wildcard-free: a group rule already covers everything nested beneath
// the path, and a trailing "/*" narrows it to the groups *below* that one instead.
export const SCOPE_TYPE_OPTIONS: ScopeTypeOption[] = [
    { value: 'WORKSPACE', label: 'Workspace', placeholder: 'workspace path glob pattern or TRN (e.g. group/prod/*)' },
    { value: 'GROUP', label: 'Group', placeholder: 'group path glob pattern or TRN (e.g. group/prod)' },
    { value: 'MANAGED_IDENTITY', label: 'Managed Identity', placeholder: 'managed identity path glob pattern or TRN (e.g. group/prod/admin-role)' },
];

export const SCOPE_TYPE_LABELS: Record<string, string> = Object.fromEntries(
    SCOPE_TYPE_OPTIONS.map(o => [o.value, o.label])
);

/**
 * scopeActionColor is the colour an action is shown in wherever scope rules are reported: green for
 * what a policy is applied to, red for what it is held away from. Shared so the counts on a policy
 * card and the rule list on the detail page cannot drift apart.
 */
export function scopeActionColor(action: string, theme: Theme): string {
    return action === 'EXCLUDE' ? theme.palette.error.main : theme.palette.success.main;
}

/** A fresh row. Every row needs its own _id, so this has to be a function rather than a constant. */
export function blankScopeRule(): ScopeRuleFormData {
    return { type: 'WORKSPACE', action: 'INCLUDE', pattern: '', _id: nanoid() };
}

/** A row the user has not finished filling in. Every rule is a pattern, so that is the only field to check. */
export function isScopeRuleComplete(rule: ScopeRuleFormData): boolean {
    return !!rule.pattern.trim();
}

/**
 * toScopeRuleInputs maps form rows to PolicyScopeRuleInput, dropping the ones with no value yet. Since
 * a row is always on screen whether or not the policy needs a scope, an unfilled row means "no rule" —
 * and an empty scope already means the policy applies to every workspace under the owning group.
 */
export function toScopeRuleInputs(scope: ScopeRuleFormData[]) {
    return scope.filter(isScopeRuleComplete).map(rule => ({
        type: rule.type,
        action: rule.action,
        pattern: rule.pattern.trim(),
    }));
}
