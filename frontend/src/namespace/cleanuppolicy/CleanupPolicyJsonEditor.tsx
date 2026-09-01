import { useEffect, useMemo, useState } from 'react';
import { Box } from '@mui/material';
import Editor from '@monaco-editor/react';
import '../../common/monaco';
import CopyButton from '../../common/CopyButton';
import { CLEANUP_POLICY_KINDS } from './rules';
import { CleanupPolicyKind, CleanupRule } from './types';

const EDITOR_HEIGHT = 600;

const VERSION_KINDS: CleanupPolicyKind[] = ['TERRAFORM_MODULES', 'TERRAFORM_PROVIDERS'];

function sharedProperties(kind: CleanupPolicyKind) {
    const isVersionKind = VERSION_KINDS.includes(kind);
    return {
        strategy: { enum: isVersionKind ? ['AGE', 'PROTECT'] : ['COUNT', 'AGE', 'PROTECT'], description: 'What happens to a match' },
        description: { type: 'string' },
        ...(!isVersionKind ? { keepMin: { type: 'integer', minimum: 0, description: 'How many of the newest matches to keep' } } : {}),
        deleteAfterDays: { type: 'integer', minimum: 0, description: 'Delete matches older than this many days' },
    };
}

const SHARED_REQUIRED = ['strategy'];

function schemaFor(kind: CleanupPolicyKind): object {
    const kindDef = CLEANUP_POLICY_KINDS[kind];
    return {
        type: 'array',
        items: {
            type: 'object',
            required: SHARED_REQUIRED,
            additionalProperties: false,
            properties: { ...sharedProperties(kind), ...kindDef.jsonSchemaProperties },
        },
    };
}

// serializeRule omits any field that matches newRule()'s default (strategy-inapplicable zeros,
// empty strings, null booleans, empty arrays), and deserializeRule fills them back in on parse.
function serializeRule(kind: CleanupPolicyKind, rule: CleanupRule): object {
    const defaults = CLEANUP_POLICY_KINDS[kind].newRule() as Record<string, unknown>;
    return Object.fromEntries(
        Object.entries(rule as unknown as Record<string, unknown>).filter(([k, v]) => JSON.stringify(v) !== JSON.stringify(defaults[k]))
    );
}

function deserializeRule(kind: CleanupPolicyKind, rule: Record<string, unknown>): CleanupRule {
    return { ...CLEANUP_POLICY_KINDS[kind].newRule(), ...rule } as CleanupRule;
}

interface Props {
    kind: CleanupPolicyKind;
    rules: readonly CleanupRule[];
    onChange: (rules: CleanupRule[]) => void;
    onValidationChange?: (hasErrors: boolean) => void;
    readOnly?: boolean;
}

function CleanupPolicyJsonEditor({ kind, rules, onChange, onValidationChange, readOnly }: Props) {
    const [text, setText] = useState(() => JSON.stringify(rules.map(r => serializeRule(kind, r)), null, 2));
    const schema = useMemo(() => schemaFor(kind), [kind]);

    useEffect(() => () => onValidationChange?.(false), [onValidationChange]);

    const onEditorChange = (value: string | undefined) => {
        const next = value ?? '';
        setText(next);
        try {
            const parsed = JSON.parse(next);
            // Anything but an array is invalid, otherwise Save would commit the last rules that parsed.
            if (!Array.isArray(parsed)) {
                onValidationChange?.(true);
                return;
            }
            onChange(parsed.map(r => deserializeRule(kind, r)));
            onValidationChange?.(false);
        } catch {
            // Invalid JSON while typing — hold current rules until it parses.
            onValidationChange?.(true);
        }
    };

    return (
        <Box>
            <Box sx={{ position: 'relative', border: '1px solid', borderColor: 'divider', borderRadius: 1, overflow: 'hidden' }}>
                <Box sx={{ position: 'absolute', top: 4, right: 4, zIndex: 1 }}>
                    <CopyButton data={text} toolTip="Copy rules as JSON" sxCopyIconStyles={{ width: 16, height: 16, color: 'common.white' }} />
                </Box>
                <Editor
                    height={EDITOR_HEIGHT}
                    defaultLanguage="json"
                    theme="vs-dark"
                    path={`tharsis://cleanup-policy/${kind}.json`}
                    value={text}
                    onChange={onEditorChange}
                    onMount={(_editor, monaco) => {
                        try {
                            const modelUri = `tharsis://cleanup-policy/${kind}.json`;
                            monaco.languages.json.jsonDefaults.setDiagnosticsOptions({
                                validate: true,
                                schemas: [{ uri: modelUri, fileMatch: [modelUri], schema }],
                            });
                        } catch {
                            // Schema validation is a nicety; the editor still works without it.
                        }
                    }}
                    options={{ fontSize: 13, minimap: { enabled: false }, scrollBeyondLastLine: false, wordWrap: 'on', accessibilitySupport: 'off', readOnly, fixedOverflowWidgets: true }}
                />
            </Box>
        </Box>
    );
}

export default CleanupPolicyJsonEditor;
