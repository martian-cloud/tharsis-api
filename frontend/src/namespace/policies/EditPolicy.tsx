import { Box, Button, Divider, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { nanoid } from 'nanoid';
import { useState } from 'react';
import { useFragment, useLazyLoadQuery, useMutation } from 'react-relay/hooks';
import { Link as RouterLink, useNavigate, useParams } from 'react-router-dom';
import { MutationError } from '../../common/error';
import NamespaceBreadcrumbs from '../NamespaceBreadcrumbs';
import { EditPolicyFragment_policy$key } from './__generated__/EditPolicyFragment_policy.graphql';
import { EditPolicyMutation } from './__generated__/EditPolicyMutation.graphql';
import { EditPolicyQuery } from './__generated__/EditPolicyQuery.graphql';
import PolicyForm, { buildApproverInput, buildKindDataInput, hasRequiredKindData, isMissingApprovers, PolicyFormData } from './PolicyForm';
import { ScopeRuleActionValue, ScopeRuleTypeValue, toScopeRuleInputs } from './scopeRules';

// The policy the form is seeded from is loaded by id rather than handed down from the list, so this
// route stands on its own when it is opened directly by url.
const query = graphql`
    query EditPolicyQuery($id: String!) {
        node(id: $id) {
            ... on Policy {
                ...EditPolicyFragment_policy
            }
        }
    }
`;

interface Props {
    ownerPath: string;
    groupPath: string;
}

function EditPolicy({ ownerPath, groupPath }: Props) {
    const { policyId } = useParams<{ policyId: string }>();
    const navigate = useNavigate();

    const queryData = useLazyLoadQuery<EditPolicyQuery>(query, { id: policyId ?? '' }, { fetchPolicy: 'store-and-network' });

    const policy = useFragment<EditPolicyFragment_policy$key>(graphql`
        fragment EditPolicyFragment_policy on Policy {
            id
            name
            description
            kind
            requiredApprovals
            opaData {
                packageSource
                packageVersionConstraint
                packageDigest
                stage
                enforcementLevel
                speculativeRunEnforcementLevel
            }
            moduleAttestationData {
                publicKey
                predicateType
                verifyStateLineage
                stage
                enforcementLevel
                speculativeRunEnforcementLevel
            }
            scope {
                type
                action
                pattern
            }
            allowedUsers { id email username }
            allowedTeams { id name }
            allowedServiceAccounts { id name resourcePath }
        }
    `, queryData.node);

    const packageSource = policy?.opaData?.packageSource ?? '';
    // The two kind-specific data objects are mutually exclusive, so whichever is present carries the
    // fields (stage, enforcement levels) that are common in shape but stored per kind.
    const kindData = policy?.opaData ?? policy?.moduleAttestationData;

    const initialFormData: PolicyFormData | null = policy
        ? {
            // The enum casts below narrow away the "%future added value" member relay adds to every
            // read type. The form's own unions are the values it can actually render, and a server
            // that starts sending something outside them needs a form change anyway.
            // A policy's type is fixed, so the form shows it read-only; it is not submitted, since
            // UpdatePolicyInput has no kind field.
            kind: policy.kind as PolicyFormData['kind'],
            name: policy.name,
            description: policy.description,
            opa: {
                // A policy stores only its package's source, so the option is synthetic: everything the
                // dropdown would show about the package is left empty and only the source is submitted.
                package: {
                    id: '',
                    label: packageSource,
                    packageSource,
                    visibility: '',
                    description: '',
                },
                packageVersionConstraint: policy.opaData?.packageVersionConstraint ?? '',
                packageDigest: policy.opaData?.packageDigest ?? '',
            },
            moduleAttestation: {
                publicKey: policy.moduleAttestationData?.publicKey ?? '',
                predicateType: policy.moduleAttestationData?.predicateType ?? '',
                verifyStateLineage: policy.moduleAttestationData?.verifyStateLineage ?? false,
            },
            stage: (kindData?.stage ?? 'POST_PLAN') as PolicyFormData['stage'],
            enforcementLevel: (kindData?.enforcementLevel ?? 'ADVISORY') as PolicyFormData['enforcementLevel'],
            speculativeRunEnforcementLevel:
                (kindData?.speculativeRunEnforcementLevel ?? 'ADVISORY') as PolicyFormData['speculativeRunEnforcementLevel'],
            requiredApprovals: policy.requiredApprovals,
            allowedUsers: (policy.allowedUsers ?? []).map(u => ({ id: u.id, email: u.email, username: u.username })),
            allowedTeams: (policy.allowedTeams ?? []).map(t => ({ id: t.id, name: t.name })),
            allowedServiceAccounts: (policy.allowedServiceAccounts ?? []).map(s => ({ id: s.id, name: s.name, resourcePath: s.resourcePath })),
            // A rule is entirely its type, action and pattern, so it round-trips as-is: what the form shows
            // is what PolicyScopeRuleInput takes back verbatim.
            scope: (policy.scope ?? []).map(r => ({
                type: r.type as ScopeRuleTypeValue,
                action: r.action as ScopeRuleActionValue,
                pattern: r.pattern,
                _id: nanoid(),
            })),
        }
        : null;

    const [formData, setFormData] = useState<PolicyFormData | null>(initialFormData);
    const [error, setError] = useState<MutationError>();

    const [commitUpdate, commitUpdateInFlight] = useMutation<EditPolicyMutation>(graphql`
        mutation EditPolicyMutation($input: UpdatePolicyInput!) {
            updatePolicy(input: $input) {
                policy {
                    id
                    name
                    description
                    kind
                    opaData {
                        packageSource
                        packageVersionConstraint
                        packageDigest
                        stage
                        enforcementLevel
                        speculativeRunEnforcementLevel
                    }
                    moduleAttestationData {
                        publicKey
                        predicateType
                        verifyStateLineage
                        stage
                        enforcementLevel
                        speculativeRunEnforcementLevel
                    }
                    createdBy
                    requiredApprovals
                    scope {
                        type
                        action
                        pattern
                    }
                    groupPath
                    allowedUsers { id email username }
                    allowedTeams { id name }
                    allowedServiceAccounts { id name resourcePath }
                }
                problems { message field type }
            }
        }
    `);

    if (!policy || !formData) {
        return null;
    }

    const onSave = () => {
        if (!hasRequiredKindData(formData) || isMissingApprovers(formData)) return;
        setError(undefined);

        const approverInput = buildApproverInput(formData);

        commitUpdate({
            variables: {
                input: {
                    id: policy.id,
                    description: formData.description || null,
                    ...buildKindDataInput(formData),
                    scope: toScopeRuleInputs(formData.scope),
                    ...approverInput,
                },
            },
            onCompleted: data => {
                if (data.updatePolicy.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.updatePolicy.problems.map(p => p.message).join('; '),
                    });
                } else {
                    navigate('..');
                }
            },
            onError: err => {
                setError({ severity: 'error', message: `Unexpected error occurred: ${err.message}` });
            },
        });
    };

    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={ownerPath}
                childRoutes={[
                    { title: 'policies', path: 'policies' },
                    { title: policy.name, path: policy.id },
                    { title: 'edit', path: 'edit' },
                ]}
            />
            <Typography variant="h5" sx={{ mb: 2 }}>Edit Policy</Typography>
            <PolicyForm
                editMode
                groupPath={groupPath}
                data={formData}
                onChange={setFormData}
                error={error}
            />
            <Divider light sx={{ marginTop: 4 }} />
            <Box marginTop={2}>
                <Button
                    loading={commitUpdateInFlight}
                    variant="outlined"
                    color="primary"
                    sx={{ marginRight: 2 }}
                    disabled={!hasRequiredKindData(formData)}
                    onClick={onSave}
                >
                    Save Changes
                </Button>
                <Button component={RouterLink} color="inherit" to={-1 as any}>Cancel</Button>
            </Box>
        </Box>
    );
}

export default EditPolicy;
