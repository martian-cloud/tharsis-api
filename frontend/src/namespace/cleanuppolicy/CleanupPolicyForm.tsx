import { useEffect, useState } from 'react';
import { Alert, Stack, Switch, Tab, Tabs, Typography } from '@mui/material';
import { MutationError } from '../../common/error';
import CleanupPolicyFormSection from './CleanupPolicyFormSection';
import CleanupPolicyJsonEditor from './CleanupPolicyJsonEditor';
import { CLEANUP_POLICY_KINDS } from './rules';
import { CleanupPolicyKind, CleanupRule } from './types';

export interface CleanupPolicyFormData {
    disabled: boolean;
    rules: readonly CleanupRule[];
}

export const DEFAULT_CLEANUP_POLICY_FORM_DATA: CleanupPolicyFormData = {
    disabled: false,
    rules: [],
};

interface Props {
    kind: CleanupPolicyKind;
    data: CleanupPolicyFormData;
    onChange: (data: CleanupPolicyFormData) => void;
    // The JSON tab can hold rules that don't parse; the Visual tab never can. Callers use this to
    // gate their own Save button, since this component has no submit action of its own.
    onValidationChange?: (hasErrors: boolean) => void;
    error?: MutationError;
}

// Settings + Rules body shared by the New and Edit cleanup policy pages. Purely a controlled form
// over CleanupPolicyFormData — creating vs. editing, the mutation that fires, and the Save/Cancel
// actions all live in the caller, the same split PolicyForm uses for policies.
function CleanupPolicyForm({ kind, data, onChange, onValidationChange, error }: Props) {
    const kindDef = CLEANUP_POLICY_KINDS[kind];

    const [adding, setAdding] = useState<CleanupRule | undefined>(undefined);
    const [jsonParseError, setJsonParseError] = useState(false);
    const [view, setView] = useState<string>('visual');

    useEffect(() => {
        onValidationChange?.(jsonParseError);
    }, [jsonParseError, onValidationChange]);

    // Leaving the JSON tab means any parse error from a half-finished edit is no longer relevant —
    // only the JSON editor can ever set jsonParseError to true, so nothing ever clears it back to
    // false once you switch away, even after fixing the rules in the Visual editor (whose onChange
    // has no reason to know about JSON-tab state).
    const handleViewChange = (_: unknown, newView: string) => {
        setView(newView);
        if (newView !== 'json') {
            setJsonParseError(false);
        }
    };

    return (
        <>
            <CleanupPolicyFormSection step={2} title="Status">
                <Typography variant="body2" color="textSecondary" sx={{ mb: 1.5 }}>
                    A {kindDef.label.toLowerCase()} sweep will not claim or delete anything while this policy is disabled.
                </Typography>
                <Stack direction="row" alignItems="center" spacing={1.5}>
                    <Switch checked={!data.disabled} onChange={e => onChange({ ...data, disabled: !e.target.checked })} />
                    <Typography variant="body2">{data.disabled ? 'Disabled' : 'Enabled'}</Typography>
                </Stack>
            </CleanupPolicyFormSection>

            <CleanupPolicyFormSection
                step={3}
                title="Rules"
                action={
                    <Tabs value={view} onChange={handleViewChange} sx={{ minHeight: 32 }}>
                        <Tab label="Visual" value="visual" sx={{ minHeight: 32, py: 0 }} />
                        <Tab label="JSON" value="json" sx={{ minHeight: 32, py: 0 }} />
                    </Tabs>
                }
            >
                {error && <Alert severity={error.severity} sx={{ mb: 2 }}>{error.message}</Alert>}
                {view === 'json'
                    ? <CleanupPolicyJsonEditor
                        kind={kind}
                        rules={data.rules}
                        onChange={rules => onChange({ ...data, rules })}
                        onValidationChange={setJsonParseError}
                    />
                    : kindDef.renderList({
                        rules: data.rules, onChange: rules => onChange({ ...data, rules }),
                        adding, onAddingChange: setAdding,
                        onRequestAdd: () => setAdding(kindDef.newRule()),
                    })
                }
            </CleanupPolicyFormSection>
        </>
    );
}

export default CleanupPolicyForm;
