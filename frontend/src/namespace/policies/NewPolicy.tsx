import { Box, Button, Divider, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useState } from 'react';
import { useMutation } from 'react-relay/hooks';
import { Link as RouterLink, useNavigate } from 'react-router-dom';
import { ConnectionHandler } from 'relay-runtime';
import { MutationError } from '../../common/error';
import NamespaceBreadcrumbs from '../NamespaceBreadcrumbs';
import PolicyForm, { buildApproverInput, DEFAULT_POLICY_FORM_DATA, isMissingApprovers, PolicyFormData } from './PolicyForm';
import { toScopeRuleInputs } from './scopeRules';
import { NewPolicyCreateMutation } from './__generated__/NewPolicyCreateMutation.graphql';

interface Props {
    ownerId: string;
    ownerPath: string;
    groupPath: string;
}

function NewPolicy({ ownerId, ownerPath, groupPath }: Props) {
    const navigate = useNavigate();
    const [formData, setFormData] = useState<PolicyFormData>(DEFAULT_POLICY_FORM_DATA);
    const [error, setError] = useState<MutationError>();

    // Reconstructed rather than handed down from PolicyList: the two views are independent routes, and
    // this way neither needs to run the other's query to know the connection's store id.
    const connectionId = ConnectionHandler.getConnectionID(
        ownerId,
        'PolicyList_policies',
        { includeInherited: true, sort: 'CREATED_AT_DESC' }
    );

    const [commitCreate, commitCreateInFlight] = useMutation<NewPolicyCreateMutation>(graphql`
        mutation NewPolicyCreateMutation($input: CreatePolicyInput!, $connections: [ID!]!) {
            createPolicy(input: $input) {
                # @prependNode puts the new policy at the top of the list connection, which is where the
                # newest-first sort would have placed it on a refetch.
                policy @prependNode(connections: $connections, edgeTypeName: "PolicyEdge") {
                    id
                    # The list reads the owning path off the node itself to decide whether the card is
                    # inherited; everything else the card shows comes from its own fragment.
                    groupPath
                    ...PolicyDetailsFragment_policy
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const onCreate = () => {
        // The kind check also narrows it to a value the mutation input accepts.
        if (!formData.kind || !formData.name || !formData.package?.packageSource || isMissingApprovers(formData)) return;
        setError(undefined);

        const approverInput = buildApproverInput(formData);

        commitCreate({
            variables: {
                input: {
                    kind: formData.kind,
                    groupId: ownerId,
                    name: formData.name,
                    description: formData.description || null,
                    opaData: {
                        packageSource: formData.package!.packageSource,
                        packageVersionConstraint: formData.packageVersionConstraint.trim() || null,
                        packageDigest: formData.packageDigest.trim() || null,
                        stage: formData.stage,
                        enforcementLevel: formData.enforcementLevel,
                        speculativeRunEnforcementLevel: formData.speculativeRunEnforcementLevel,
                    },
                    scope: toScopeRuleInputs(formData.scope),
                    ...approverInput,
                },
                connections: [connectionId],
            },
            onCompleted: data => {
                if (data.createPolicy.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.createPolicy.problems.map(p => p.message).join('; '),
                    });
                } else if (!data.createPolicy.policy) {
                    setError({ severity: 'error', message: 'Unexpected error occurred' });
                } else {
                    navigate(`../${data.createPolicy.policy.id}`);
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
                    { title: 'new', path: 'new' },
                ]}
            />
            <Typography variant="h5" mb={2}>New Policy</Typography>
            <PolicyForm
                groupPath={groupPath}
                data={formData}
                onChange={setFormData}
                error={error}
            />
            <Divider light sx={{ marginTop: 4 }} />
            <Box marginTop={2}>
                <Button
                    loading={commitCreateInFlight}
                    disabled={!formData.kind || !formData.name || !formData.package?.packageSource || isMissingApprovers(formData)}
                    variant="outlined"
                    color="primary"
                    sx={{ marginRight: 2 }}
                    onClick={onCreate}
                >
                    Add Policy
                </Button>
                <Button component={RouterLink} color="inherit" to={-1 as any}>Cancel</Button>
            </Box>
        </Box>
    );
}

export default NewPolicy;
