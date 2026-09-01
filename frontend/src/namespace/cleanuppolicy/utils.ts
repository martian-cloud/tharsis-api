/**
 * utils.ts — pure helpers shared across the cleanup policy components.
 */
import { CLEANUP_POLICY_KINDS } from './rules';
import { CleanupPolicyKind, CleanupRule, PolicyDataKey } from './types';

// policyDataInput builds the per-kind data payload for create/update mutations, setting
// the active kind's rules and nulling out all others.
export function policyDataInput(kind: CleanupPolicyKind, rules: readonly CleanupRule[]): Record<string, any> {
    return Object.fromEntries(
        (Object.keys(CLEANUP_POLICY_KINDS) as CleanupPolicyKind[]).map(k => [
            CLEANUP_POLICY_KINDS[k].policyDataKey,
            k === kind ? { rules } : null,
        ])
    );
}

// moveRule swaps index with index+offset. Order determines which rule wins a match.
export function moveRule<R extends CleanupRule>(rules: readonly R[], index: number, offset: number): readonly R[] {
    const target = index + offset;
    if (target < 0 || target >= rules.length) return rules;
    const reordered = [...rules];
    [reordered[index], reordered[target]] = [reordered[target], reordered[index]];
    return reordered;
}

// policyRules extracts the rules array from a policy's data wrapper for a given kind.
export function policyRules(
    policy: { readonly [K in PolicyDataKey]?: { readonly rules: readonly unknown[] } | null } | null | undefined,
    key: PolicyDataKey,
): readonly CleanupRule[] {
    return (policy?.[key]?.rules ?? []) as readonly CleanupRule[];
}

// JSON.stringify is key-order sensitive, so sort keys before comparing rules to server rules.
export function stableStringify(value: unknown): string {
    if (Array.isArray(value)) return `[${value.map(stableStringify).join(',')}]`;
    if (value !== null && typeof value === 'object') {
        const entries = Object.entries(value as Record<string, unknown>).sort(([a], [b]) => a.localeCompare(b));
        return `{${entries.map(([k, v]) => `${JSON.stringify(k)}:${stableStringify(v)}`).join(',')}}`;
    }
    return JSON.stringify(value);
}
