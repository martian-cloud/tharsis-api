/**
 * types.ts — cleanup policy rule shapes shared by the registry, the editors and the rule list.
 */

export type CleanupStrategy = 'COUNT' | 'AGE' | 'PROTECT';

export type CleanupPolicyKind = 'TERRAFORM_MODULES' | 'TERRAFORM_PROVIDERS' | 'RUNS';

// The field holding a kind's policy data on the policy type and the mutation inputs.
export type PolicyDataKey = 'terraformModulePolicyData' | 'terraformProviderPolicyData' | 'runPolicyData';

export interface TerraformModuleCleanupRule {
    strategy: CleanupStrategy;
    description: string;
    nameGlob: string;
    systemGlob: string;
    versionGlob: string;
    deleteAfterDays: number;
}

export interface TerraformProviderCleanupRule {
    strategy: CleanupStrategy;
    description: string;
    nameGlob: string;
    versionGlob: string;
    deleteAfterDays: number;
}

export interface RunCleanupRule {
    strategy: CleanupStrategy;
    description: string;
    speculative: boolean | null;
    assessment: boolean | null;
    status: string[];
    keepMin: number;
    deleteAfterDays: number;
}

export type CleanupRule = TerraformModuleCleanupRule | TerraformProviderCleanupRule | RunCleanupRule;

// Condition is one qualifier a rule's "applies to" is built from. label is the prose ("name
// matches"); value is the literal glob/status shown as code so it reads as syntax
// rather than English. A boolean run-kind flag has no value of its own.
export interface Condition {
    label: string;
    value?: string;
}

export interface CleanupKindPanelProps<R extends CleanupRule> {
    rules: readonly R[];
    onChange?: (rules: readonly R[]) => void;
    adding: R | undefined;
    onAddingChange: (rule: R | undefined) => void;
    onRequestAdd: () => void;
}
