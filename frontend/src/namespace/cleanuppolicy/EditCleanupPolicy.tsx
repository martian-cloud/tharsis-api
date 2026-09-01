import { useMemo, useState } from 'react';
import { Alert, Box, Button, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useLazyLoadQuery, useMutation } from 'react-relay/hooks';
import { Link as RouterLink, useNavigate, useParams } from 'react-router-dom';
import { useSnackbar } from 'notistack';
import ConfirmationDialog from '../../common/ConfirmationDialog';
import { MutationError } from '../../common/error';
import NamespaceBreadcrumbs from '../NamespaceBreadcrumbs';
import CleanupPolicyForm, { CleanupPolicyFormData } from './CleanupPolicyForm';
import CleanupPolicyFormActions from './CleanupPolicyFormActions';
import CleanupPolicyFormSection from './CleanupPolicyFormSection';
import CleanupPolicyKindSelector from './CleanupPolicyKindSelector';
import { CLEANUP_POLICY_KINDS } from './rules';
import { CleanupPolicyKind } from './types';
import { policyDataInput, policyRules, stableStringify } from './utils';
import { EditCleanupPolicyQuery } from './__generated__/EditCleanupPolicyQuery.graphql';
import { EditCleanupPolicyDeleteMutation } from './__generated__/EditCleanupPolicyDeleteMutation.graphql';
import { EditCleanupPolicyUpdateMutation } from './__generated__/EditCleanupPolicyUpdateMutation.graphql';

const query = graphql`
    query EditCleanupPolicyQuery($namespacePath: String!) {
        namespace(fullPath: $namespacePath) {
            id
            __typename
            fullPath
            effectiveCleanupPolicies {
                id
                kind
                namespacePath
                disabled
                lastSweepCompletedAt
                terraformModulePolicyData {
                    rules {
                        strategy
                        description
                        nameGlob
                        systemGlob
                        versionGlob
                        deleteAfterDays
                    }
                }
                terraformProviderPolicyData {
                    rules {
                        strategy
                        description
                        nameGlob
                        versionGlob
                        deleteAfterDays
                    }
                }
                runPolicyData {
                    rules {
                        strategy
                        description
                        speculative
                        assessment
                        status
                        keepMin
                        deleteAfterDays
                    }
                }
            }
        }
    }
`;

interface ContentProps {
    kind: CleanupPolicyKind;
    namespacePath: string;
}

function EditCleanupPolicyContent({ kind, namespacePath }: ContentProps) {
    const navigate = useNavigate();
    const { enqueueSnackbar } = useSnackbar();

    const kindDef = CLEANUP_POLICY_KINDS[kind];

    const queryData = useLazyLoadQuery<EditCleanupPolicyQuery>(
        query,
        { namespacePath },
        { fetchPolicy: 'store-and-network' }
    );

    // __typename is fetched by Relay for abstract-type normalization even though it's not in
    // the TypeScript type; safe to access at runtime.
    const isWorkspace = (queryData.namespace as any)?.__typename === 'Workspace';
    const kinds = useMemo(
        () => (Object.keys(CLEANUP_POLICY_KINDS) as CleanupPolicyKind[])
            .filter(k => !isWorkspace || !CLEANUP_POLICY_KINDS[k].groupOnly),
        [isWorkspace],
    );

    // The type can never change once a policy exists, so every kind other than the one being
    // edited is disabled here, each with a tooltip explaining why.
    const disabledKinds = useMemo(() => {
        const reasons = new Map<CleanupPolicyKind, string>();
        kinds.forEach(k => {
            if (k !== kind) {
                reasons.set(k, `This policy's type can't be changed. Delete it and create a new ${CLEANUP_POLICY_KINDS[k].label} policy instead.`);
            }
        });
        return reasons;
    }, [kinds, kind]);

    // localPolicy is the policy owned by this namespace (not inherited).
    const localPolicy = queryData.namespace?.effectiveCleanupPolicies?.find(
        (p: any) => p.kind === kind && p.namespacePath === namespacePath
    );
    const serverRules = policyRules(localPolicy ?? undefined, kindDef.policyDataKey);
    const initialFormData: CleanupPolicyFormData = { disabled: localPolicy?.disabled ?? false, rules: serverRules };

    const [formData, setFormData] = useState<CleanupPolicyFormData>(initialFormData);
    const [jsonInvalid, setJsonInvalid] = useState(false);
    const [saveError, setSaveError] = useState<MutationError>();
    const [deleteError, setDeleteError] = useState<MutationError>();
    const [confirmDelete, setConfirmDelete] = useState(false);

    // Only enables Save once something actually changed from what the server has.
    const dirty = stableStringify(formData.rules) !== stableStringify(serverRules)
        || formData.disabled !== (localPolicy?.disabled ?? false);

    const [commitUpdate, updateInFlight] = useMutation<EditCleanupPolicyUpdateMutation>(graphql`
        mutation EditCleanupPolicyUpdateMutation($input: UpdateCleanupPolicyInput!) {
            updateCleanupPolicy(input: $input) {
                namespace {
                    effectiveCleanupPolicies {
                        ...CleanupPolicyListItem_fields
                    }
                }
                problems { message field type }
            }
        }
    `);

    const [commitDelete, deleteInFlight] = useMutation<EditCleanupPolicyDeleteMutation>(graphql`
        mutation EditCleanupPolicyDeleteMutation($input: DeleteCleanupPolicyInput!) {
            deleteCleanupPolicy(input: $input) {
                namespace {
                    effectiveCleanupPolicies {
                        ...CleanupPolicyListItem_fields
                    }
                }
                problems { message field type }
            }
        }
    `);

    const onSave = () => {
        if (!localPolicy?.id) return;
        commitUpdate({
            variables: { input: { id: localPolicy.id, disabled: formData.disabled, ...policyDataInput(kind, formData.rules) } },
            onCompleted: data => {
                const problems = data.updateCleanupPolicy.problems;
                if (problems.length) {
                    setSaveError({ severity: 'warning', message: problems.map(p => p.message).join('; ') });
                    return;
                }
                navigate('..');
                enqueueSnackbar(`${kindDef.label} policy saved`, { variant: 'success' });
            },
            onError: failure => setSaveError({ severity: 'error', message: failure.message }),
        });
    };

    const onDelete = () => {
        if (!localPolicy?.id) return;
        commitDelete({
            variables: { input: { id: localPolicy.id } },
            onCompleted: data => {
                const problems = data.deleteCleanupPolicy.problems;
                if (problems.length) {
                    setDeleteError({ severity: 'warning', message: problems.map(p => p.message).join('; ') });
                    return;
                }
                navigate('..');
                enqueueSnackbar(`${kindDef.label} policy deleted`, { variant: 'success' });
            },
            onError: failure => setDeleteError({ severity: 'error', message: failure.message }),
        });
    };

    const breadcrumbs = (
        <NamespaceBreadcrumbs
            namespacePath={namespacePath}
            childRoutes={[
                { title: 'cleanup policies', path: 'cleanup_policies' },
                { title: kindDef.label, path: kind.toLowerCase() },
            ]}
        />
    );

    if (!localPolicy) {
        return (
            <Box>
                {breadcrumbs}
                <Alert severity="info" sx={{ mt: 2 }}
                    action={
                        <Button color="inherit" size="small" component={RouterLink} to={`../new?kind=${kind.toLowerCase()}`}>
                            Create policy
                        </Button>
                    }
                >
                    No local {kindDef.label.toLowerCase()} policy exists for this namespace.
                </Alert>
            </Box>
        );
    }

    return (
        <Box>
            {breadcrumbs}

            <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', mb: 3, flexWrap: 'wrap', gap: 1 }}>
                <Box>
                    <Typography variant="h5" gutterBottom>Edit {kindDef.label} Policy</Typography>
                    <Typography variant="body2" color="textSecondary">{kindDef.description}</Typography>
                </Box>
                <Button
                    size="small" variant="outlined" color="error"
                    disabled={deleteInFlight}
                    onClick={() => setConfirmDelete(true)}
                    sx={{ flexShrink: 0 }}
                >
                    Delete policy
                </Button>
            </Box>

            <CleanupPolicyFormSection step={1} title="Type">
                {/* No onSelect — the kind can't change once a policy exists — and every other kind
                    is passed via disabledKinds so it's visibly disabled with an explanation,
                    rather than just inert. */}
                <CleanupPolicyKindSelector kinds={kinds} selectedKind={kind} disabledKinds={disabledKinds} />
            </CleanupPolicyFormSection>

            <CleanupPolicyForm
                key={localPolicy.id}
                kind={kind}
                data={formData}
                onChange={setFormData}
                onValidationChange={setJsonInvalid}
                error={saveError}
            />
            <CleanupPolicyFormActions
                primaryLabel="Save changes"
                onPrimaryClick={onSave}
                primaryDisabled={!dirty || jsonInvalid}
                primaryLoading={updateInFlight}
                cancelDisabled={updateInFlight || deleteInFlight}
            />

            {deleteError && <Alert severity={deleteError.severity} sx={{ mt: 2 }}>{deleteError.message}</Alert>}

            {confirmDelete && (
                <ConfirmationDialog
                    title={`Delete ${kindDef.label} Policy`}
                    confirmLabel="Delete"
                    confirmInProgress={deleteInFlight}
                    onConfirm={() => {
                        setConfirmDelete(false);
                        onDelete();
                    }}
                    onClose={() => setConfirmDelete(false)}
                >
                    Are you sure you want to delete the <strong>{kindDef.label}</strong> cleanup policy? All rules will be lost.
                </ConfirmationDialog>
            )}
        </Box>
    );
}

interface Props {
    namespacePath: string;
}

function EditCleanupPolicy({ namespacePath }: Props) {
    const { kind: kindParam } = useParams<{ kind: string }>();

    const kind = kindParam?.toUpperCase() as CleanupPolicyKind;

    if (!CLEANUP_POLICY_KINDS[kind]) {
        return (
            <Box display="flex" justifyContent="center" mt={4}>
                <Typography color="textSecondary">Policy kind not found.</Typography>
            </Box>
        );
    }

    return <EditCleanupPolicyContent kind={kind} namespacePath={namespacePath} />;
}

export default EditCleanupPolicy;
