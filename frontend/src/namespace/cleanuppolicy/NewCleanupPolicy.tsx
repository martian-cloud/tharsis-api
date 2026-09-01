import { useMemo, useState } from 'react';
import { Box, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useLazyLoadQuery, useMutation } from 'react-relay/hooks';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useSnackbar } from 'notistack';
import { MutationError } from '../../common/error';
import NamespaceBreadcrumbs from '../NamespaceBreadcrumbs';
import CleanupPolicyForm, { CleanupPolicyFormData, DEFAULT_CLEANUP_POLICY_FORM_DATA } from './CleanupPolicyForm';
import CleanupPolicyFormActions from './CleanupPolicyFormActions';
import CleanupPolicyFormSection from './CleanupPolicyFormSection';
import CleanupPolicyKindSelector from './CleanupPolicyKindSelector';
import { CLEANUP_POLICY_KINDS } from './rules';
import { CleanupPolicyKind } from './types';
import { policyDataInput } from './utils';
import { NewCleanupPolicyQuery } from './__generated__/NewCleanupPolicyQuery.graphql';
import { NewCleanupPolicyCreateMutation } from './__generated__/NewCleanupPolicyCreateMutation.graphql';

const query = graphql`
    query NewCleanupPolicyQuery($namespacePath: String!) {
        namespace(fullPath: $namespacePath) {
            id
            __typename
            fullPath
            effectiveCleanupPolicies {
                id
                kind
                namespacePath
            }
        }
    }
`;

interface FormProps {
    kind: CleanupPolicyKind;
    namespacePath: string;
}

// Owns the create mutation and form state for one selected kind — kept separate from
// NewCleanupPolicy (and keyed by kind at the call site) so switching the selected type remounts
// with a fresh, unsaved form instead of carrying over rules from the previous kind.
function NewCleanupPolicyForm({ kind, namespacePath }: FormProps) {
    const navigate = useNavigate();
    const { enqueueSnackbar } = useSnackbar();
    const kindDef = CLEANUP_POLICY_KINDS[kind];

    const [formData, setFormData] = useState<CleanupPolicyFormData>(DEFAULT_CLEANUP_POLICY_FORM_DATA);
    const [jsonInvalid, setJsonInvalid] = useState(false);
    const [error, setError] = useState<MutationError>();

    const [commitCreate, inFlight] = useMutation<NewCleanupPolicyCreateMutation>(graphql`
        mutation NewCleanupPolicyCreateMutation($input: CreateCleanupPolicyInput!) {
            createCleanupPolicy(input: $input) {
                namespace {
                    effectiveCleanupPolicies {
                        ...CleanupPolicyListItem_fields
                    }
                }
                problems { message field type }
            }
        }
    `);

    const onCreate = () => {
        commitCreate({
            variables: { input: { namespacePath, kind, disabled: formData.disabled, ...policyDataInput(kind, formData.rules) } },
            onCompleted: data => {
                const problems = data.createCleanupPolicy.problems;
                if (problems.length) {
                    setError({ severity: 'warning', message: problems.map(p => p.message).join('; ') });
                    return;
                }
                navigate('..');
                enqueueSnackbar(`${kindDef.label} policy created`, { variant: 'success' });
            },
            onError: failure => setError({ severity: 'error', message: failure.message }),
        });
    };

    return (
        <>
            <CleanupPolicyForm
                kind={kind}
                data={formData}
                onChange={setFormData}
                onValidationChange={setJsonInvalid}
                error={error}
            />
            <CleanupPolicyFormActions
                primaryLabel="Create policy"
                onPrimaryClick={onCreate}
                primaryDisabled={jsonInvalid}
                primaryLoading={inFlight}
                cancelDisabled={inFlight}
            />
        </>
    );
}

interface Props {
    namespacePath: string;
}

function NewCleanupPolicy({ namespacePath }: Props) {
    const [searchParams] = useSearchParams();

    const queryData = useLazyLoadQuery<NewCleanupPolicyQuery>(
        query,
        { namespacePath },
        { fetchPolicy: 'store-and-network' }
    );

    // __typename is fetched by Relay for abstract-type normalization even though it's not in
    // the TypeScript type; safe to access at runtime.
    const isWorkspace = (queryData.namespace as any)?.__typename === 'Workspace';

    // All kinds valid for this namespace type are always shown — whether or not they already have
    // a policy — so the form communicates the full picture instead of silently hiding options.
    const kinds = useMemo(
        () => (Object.keys(CLEANUP_POLICY_KINDS) as CleanupPolicyKind[])
            .filter(k => !isWorkspace || !CLEANUP_POLICY_KINDS[k].groupOnly),
        [isWorkspace],
    );

    // The only place that links here with ?kind= is the list page's "Override Policy" button, so
    // its presence (and validity) means this whole page is locked to overriding that one kind —
    // the type can't be changed away from it, the same way it can't be changed on the edit page.
    const overrideKind = useMemo(() => {
        const paramKind = searchParams.get('kind')?.toUpperCase() as CleanupPolicyKind | undefined;
        return paramKind && kinds.includes(paramKind) ? paramKind : undefined;
    }, [searchParams, kinds]);

    // A kind is disabled (but still visible, with an explanation) if:
    //  - we're overriding a specific kind, in which case every *other* kind is off the table, or
    //  - otherwise, it already has any active policy here (local or inherited-only) — a local
    //    policy already has its own Edit page, and an inherited-only one already has the
    //    "Override Policy" button on the list page, so this generic form doesn't duplicate either.
    const disabledKinds = useMemo(() => {
        const reasons = new Map<CleanupPolicyKind, string>();
        kinds.forEach(k => {
            if (overrideKind) {
                if (k !== overrideKind) {
                    reasons.set(k, `Only overriding the ${CLEANUP_POLICY_KINDS[overrideKind].label} policy is available from this link.`);
                }
                return;
            }
            const policy = queryData.namespace?.effectiveCleanupPolicies?.find((p: any) => p.kind === k);
            if (!policy) return;
            reasons.set(k, policy.namespacePath === namespacePath
                ? 'A policy for this type already exists in this namespace. Edit it from the list instead.'
                : 'This type already has an inherited policy. Use Override Policy from the list to create one here.');
        });
        return reasons;
    }, [kinds, overrideKind, queryData.namespace, namespacePath]);

    const [selectedKind, setSelectedKind] = useState<CleanupPolicyKind | ''>(
        () => overrideKind ?? ''
    );

    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={namespacePath}
                childRoutes={[
                    { title: 'cleanup policies', path: 'cleanup_policies' },
                    { title: 'new', path: 'new' },
                ]}
            />

            <Box sx={{ mb: 3 }}>
                <Typography variant="h5" gutterBottom>New Cleanup Policy</Typography>
                <Typography variant="body2" color="textSecondary">
                    Choose a resource type, then configure the rules that decide what gets cleaned up.
                </Typography>
            </Box>

            <CleanupPolicyFormSection step={1} title="Type">
                <CleanupPolicyKindSelector
                    kinds={kinds}
                    selectedKind={selectedKind}
                    // Overriding locks the type entirely — omitting onSelect makes the grid fully
                    // read-only (same treatment as the edit page), rather than merely disabling
                    // the other cards while technically still allowing a change.
                    onSelect={overrideKind ? undefined : setSelectedKind}
                    disabledKinds={disabledKinds}
                />
            </CleanupPolicyFormSection>

            {selectedKind && (
                <NewCleanupPolicyForm key={selectedKind} kind={selectedKind} namespacePath={namespacePath} />
            )}
        </Box>
    );
}

export default NewCleanupPolicy;
