import { Box, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useFragment, useLazyLoadQuery } from 'react-relay/hooks';
import NoResults from '../../../common/NoResults';
import PolicyCard from '../../../namespace/policies/PolicyCard';
import { ManagedIdentityPoliciesFragment_managedIdentity$key } from './__generated__/ManagedIdentityPoliciesFragment_managedIdentity.graphql';
import { ManagedIdentityPoliciesPoliciesFragment_managedIdentity$key } from './__generated__/ManagedIdentityPoliciesPoliciesFragment_managedIdentity.graphql';
import { ManagedIdentityPoliciesQuery } from './__generated__/ManagedIdentityPoliciesQuery.graphql';
import { StyledCode } from '@/common/StyledCode';

// The policies are fetched by their own query rather than through the details query, so opening the
// identity page does not pay for a tab that may never be looked at.
const query = graphql`
    query ManagedIdentityPoliciesQuery($id: String!) {
        node(id: $id) {
            ...ManagedIdentityPoliciesPoliciesFragment_managedIdentity
        }
    }
`;

interface Props {
    fragmentRef: ManagedIdentityPoliciesFragment_managedIdentity$key;
}

function ManagedIdentityPolicies({ fragmentRef }: Props) {
    const managedIdentity = useFragment<ManagedIdentityPoliciesFragment_managedIdentity$key>(
        graphql`
        fragment ManagedIdentityPoliciesFragment_managedIdentity on ManagedIdentity {
            id
        }
        `, fragmentRef);

    const queryData = useLazyLoadQuery<ManagedIdentityPoliciesQuery>(
        query,
        { id: managedIdentity.id },
        { fetchPolicy: 'store-and-network' },
    );

    const data = useFragment<ManagedIdentityPoliciesPoliciesFragment_managedIdentity$key>(graphql`
        fragment ManagedIdentityPoliciesPoliciesFragment_managedIdentity on ManagedIdentity {
            referencingPolicies {
                id
                ...PolicyCardFragment_policy
            }
        }
    `, queryData.node);

    const policies = data?.referencingPolicies ?? [];

    return (
        <Box>
            <Typography variant="h6" gutterBottom>Referencing Policies</Typography>
            <Typography color="textSecondary" sx={{ mb: 3 }}>
                These policies reference this managed identity in a <StyledCode>Managed Identity</StyledCode> scope rule.
            </Typography>
            {policies.length === 0 && (
                <NoResults>No policies reference this managed identity.</NoResults>
            )}
            {policies.length > 0 && (
                <Box
                    sx={(theme) => ({
                        display: 'grid',
                        gridTemplateColumns: 'repeat(2, 1fr)',
                        gap: 2,
                        [theme.breakpoints.down('md')]: {
                            gridTemplateColumns: '1fr',
                        },
                    })}
                >
                    {policies.map(policy => (
                        <PolicyCard
                            key={policy.id}
                            fragmentRef={policy}
                            showGroupPath
                            showActions={false}
                        />
                    ))}
                </Box>
            )}
        </Box>
    );
}

export default ManagedIdentityPolicies;
