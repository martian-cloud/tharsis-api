import { Box, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useFragment, useLazyLoadQuery } from 'react-relay/hooks';
import NamespaceBreadcrumbs from '../namespace/NamespaceBreadcrumbs';
import PolicyCard from '../namespace/policies/PolicyCard';
import { AssignedPolicyListFragment_workspace$key } from './__generated__/AssignedPolicyListFragment_workspace.graphql';
import { AssignedPolicyListQuery } from './__generated__/AssignedPolicyListQuery.graphql';
import { AssignedPolicyListPoliciesFragment_workspace$key } from './__generated__/AssignedPolicyListPoliciesFragment_workspace.graphql';
import NoResults from '@/common/NoResults';

const query = graphql`
    query AssignedPolicyListQuery($id: String!) {
        node(id: $id) {
            ...AssignedPolicyListPoliciesFragment_workspace
        }
    }
`;

interface Props {
    fragmentRef: AssignedPolicyListFragment_workspace$key;
}

function AssignedPolicyList(props: Props) {
    const workspace = useFragment<AssignedPolicyListFragment_workspace$key>(graphql`
        fragment AssignedPolicyListFragment_workspace on Workspace {
            id
            fullPath
        }
    `, props.fragmentRef);

    const queryData = useLazyLoadQuery<AssignedPolicyListQuery>(
        query,
        { id: workspace.id },
        { fetchPolicy: 'store-and-network' },
    );

    const data = useFragment<AssignedPolicyListPoliciesFragment_workspace$key>(graphql`
        fragment AssignedPolicyListPoliciesFragment_workspace on Workspace {
            assignedPolicies {
                id
                ...PolicyCardFragment_policy
            }
        }
    `, queryData.node);

    const policies = data?.assignedPolicies ?? [];

    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={workspace.fullPath}
                childRoutes={[{ title: 'policies', path: 'policies' }]}
            />
            <Typography variant="h5" gutterBottom>Assigned Policies</Typography>
            <Typography color="textSecondary" sx={{ mb: 3 }}>
                These policies apply to runs in this workspace based on their scope rules.
            </Typography>
            {policies.length === 0 && (
                <NoResults>No policies are assigned to this workspace.</NoResults>
            )}
            {policies.length > 0 && (
                <Box
                    sx={(theme) => ({
                        display: 'grid',
                        gridTemplateColumns: 'repeat(3, 1fr)',
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

export default AssignedPolicyList;
