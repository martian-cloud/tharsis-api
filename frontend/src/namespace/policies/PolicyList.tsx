import { Box, Button, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import { useState } from 'react';
import InfiniteScroll from 'react-infinite-scroll-component';
import { useLazyLoadQuery, useMutation, usePaginationFragment } from 'react-relay/hooks';
import { Link as RouterLink } from 'react-router-dom';
import ConfirmationDialog from '../../common/ConfirmationDialog';
import ListSkeleton from '../../skeletons/ListSkeleton';
import NamespaceBreadcrumbs from '../NamespaceBreadcrumbs';
import PolicyCard from './PolicyCard';
import { PolicyListDeleteMutation } from './__generated__/PolicyListDeleteMutation.graphql';
import { PolicyListFragment_policies$key } from './__generated__/PolicyListFragment_policies.graphql';
import { PolicyListPaginationQuery } from './__generated__/PolicyListPaginationQuery.graphql';
import { PolicyListQuery } from './__generated__/PolicyListQuery.graphql';

const DESCRIPTION = `Define rules that govern infrastructure deployments. When a policy's conditions are met, it can block the deployment, require an approval, or surface advisory information`;
const INITIAL_ITEM_COUNT = 25;
const LOAD_MORE_COUNT = 25;

const query = graphql`
    query PolicyListQuery($first: Int!, $after: String, $id: String!) {
        node(id: $id) {
            ...PolicyListFragment_policies
        }
    }
`;

// policyToDelete is the policy awaiting delete confirmation.
interface DeleteTarget {
    id: string;
    name: string;
}

interface Props {
    // The group the list is rooted at. The query loads by id; ownerPath and currentGroupPath are
    // otherwise identical for a group's own policies page — a workspace's assigned-policies view is the
    // only caller where they would differ, and this list is only ever mounted under a group.
    ownerId: string;
    ownerPath: string;
}

function PolicyList({ ownerId, ownerPath }: Props) {
    const theme = useTheme();
    const { enqueueSnackbar } = useSnackbar();
    const [policyToDelete, setPolicyToDelete] = useState<DeleteTarget | null>(null);

    const queryData = useLazyLoadQuery<PolicyListQuery>(query, { first: INITIAL_ITEM_COUNT, id: ownerId }, { fetchPolicy: 'store-and-network' });

    const { data, loadNext, hasNext } = usePaginationFragment<PolicyListPaginationQuery, PolicyListFragment_policies$key>(
        graphql`
        fragment PolicyListFragment_policies on Group
        @refetchable(queryName: "PolicyListPaginationQuery") {
            policies(first: $first, after: $after, includeInherited: true, sort: CREATED_AT_DESC) @connection(key: "PolicyList_policies") {
                # The store id @deleteEdge needs to drop a deleted policy's card from the list.
                __id
                totalCount
                edges {
                    node {
                        id
                        # Everything the cards render comes from their own fragment. The detail and edit
                        # routes load their policy by id, so nothing they need is selected here.
                        groupPath
                        ...PolicyCardFragment_policy
                    }
                }
            }
        }
        `, queryData.node);

    const [commitDelete, deleteInFlight] = useMutation<PolicyListDeleteMutation>(graphql`
        mutation PolicyListDeleteMutation($input: DeletePolicyInput!, $connections: [ID!]!) {
            deletePolicy(input: $input) {
                policy {
                    id @deleteEdge(connections: $connections)
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const connectionId = data?.policies?.__id;
    const policies = (data?.policies?.edges ?? [])
        .map(edge => edge?.node)
        .filter((node): node is NonNullable<typeof node> => node != null);
    const totalCount = data?.policies?.totalCount ?? 0;

    const onDelete = (confirm?: boolean) => {
        if (!confirm || !policyToDelete) {
            setPolicyToDelete(null);
            return;
        }

        commitDelete({
            variables: {
                input: { id: policyToDelete.id },
                // @deleteEdge drops the card from the grid, so no re-read is needed.
                connections: connectionId ? [connectionId] : [],
            },
            onCompleted: response => {
                setPolicyToDelete(null);
                if (response.deletePolicy.problems.length) {
                    enqueueSnackbar(
                        response.deletePolicy.problems.map(problem => problem.message).join('; '),
                        { variant: 'warning' }
                    );
                }
            },
            onError: error => {
                setPolicyToDelete(null);
                enqueueSnackbar(`Unexpected error occurred: ${error.message}`, { variant: 'error' });
            },
        });
    };

    // totalCount is fixed when the query loads, so it cannot be the only test: deleting the last policy
    // drops its edge without changing the count. The edges are what say the list is now empty and the
    // onboarding view belongs on screen.
    const hasPolicies = totalCount > 0 && policies.length > 0;

    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={ownerPath}
                childRoutes={[{ title: 'policies', path: 'policies' }]}
            />
            {hasPolicies && (
                <Box>
                    <Box sx={{
                        display: 'flex',
                        flexDirection: 'row',
                        justifyContent: 'space-between',
                        mb: 3,
                        [theme.breakpoints.down('md')]: {
                            flexDirection: 'column',
                            alignItems: 'flex-start',
                            '& > *': { marginBottom: 2 },
                        },
                    }}>
                        <Box>
                            <Typography variant="h5" gutterBottom>Policies</Typography>
                            <Typography variant="body2">{DESCRIPTION}</Typography>
                        </Box>
                        <Box>
                            <Button sx={{ minWidth: 160 }} component={RouterLink} variant="outlined" to="new">
                                Add Policy
                            </Button>
                        </Box>
                    </Box>
                    <InfiniteScroll
                        dataLength={policies.length}
                        next={() => loadNext(LOAD_MORE_COUNT)}
                        hasMore={hasNext}
                        loader={<ListSkeleton rowCount={3} />}
                    >
                        <Box
                            sx={{
                                display: 'grid',
                                // minmax(0, 1fr) rather than 1fr: a plain fr track floors at auto, so a
                                // card holding something unbreakable — a long scope path — would widen its
                                // own column past an even share and let the text escape the card.
                                gridTemplateColumns: 'repeat(3, minmax(0, 1fr))',
                                gap: 2,
                                [theme.breakpoints.down('md')]: {
                                    gridTemplateColumns: 'minmax(0, 1fr)',
                                },
                            }}
                        >
                            {policies.map(policy => (
                                <PolicyCard
                                    key={policy.id}
                                    fragmentRef={policy}
                                    showGroupPath={policy.groupPath !== ownerPath}
                                    onDelete={setPolicyToDelete}
                                />
                            ))}
                        </Box>
                    </InfiniteScroll>
                </Box>
            )}
            {!hasPolicies && (
                <Box sx={{ marginTop: 4 }} display="flex" justifyContent="center">
                    <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center" sx={{ maxWidth: 600 }}>
                        <Typography variant="h6">Get started with policies</Typography>
                        <Typography color="textSecondary" align="center" sx={{ marginBottom: 2 }}>
                            {DESCRIPTION}
                        </Typography>
                        <Button component={RouterLink} variant="outlined" to="new">
                            Add Policy
                        </Button>
                    </Box>
                </Box>
            )}
            {policyToDelete && <ConfirmationDialog
                title="Delete Policy"
                confirmLabel="Delete"
                confirmInProgress={deleteInFlight}
                onConfirm={() => onDelete(true)}
                onClose={() => onDelete()}
            >
                Are you sure you want to delete policy <strong>{policyToDelete.name}</strong>?
            </ConfirmationDialog>}
        </Box>
    );
}

export default PolicyList;
