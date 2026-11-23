import { Box, List, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import InfiniteScroll from 'react-infinite-scroll-component';
import { useLazyLoadQuery, usePaginationFragment } from 'react-relay/hooks';
import ListSkeleton from '../../skeletons/ListSkeleton';
import VCSProviderWorkspaceListItem from './VCSProviderWorkspaceListItem';
import { VCSProviderWorkspaceListFragment_workspaces$key } from './__generated__/VCSProviderWorkspaceListFragment_workspaces.graphql';
import { VCSProviderWorkspaceListPaginationQuery } from './__generated__/VCSProviderWorkspaceListPaginationQuery.graphql';
import { VCSProviderWorkspaceListQuery } from './__generated__/VCSProviderWorkspaceListQuery.graphql';

const INITIAL_ITEM_COUNT = 50;
const LOAD_MORE_COUNT = 20;

interface Props {
    vcsProviderId: string
}

function VCSProviderWorkspaceList({ vcsProviderId }: Props) {
    const queryData = useLazyLoadQuery<VCSProviderWorkspaceListQuery>(graphql`
        query VCSProviderWorkspaceListQuery($first: Int, $after: String, $vcsProviderId: String!) {
            node(id: $vcsProviderId) {
                ...on VCSProvider @alias(as: "vcsProvider") {
                    ...VCSProviderWorkspaceListFragment_workspaces
                }
            }
        }
    `, { first: INITIAL_ITEM_COUNT, vcsProviderId }, { fetchPolicy: 'store-and-network' });

    const { data, loadNext, hasNext } = usePaginationFragment<VCSProviderWorkspaceListPaginationQuery, VCSProviderWorkspaceListFragment_workspaces$key>(
        graphql`
        fragment VCSProviderWorkspaceListFragment_workspaces on VCSProvider
        @refetchable(queryName: "VCSProviderWorkspaceListPaginationQuery") {
            workspaces(
                after: $after
                first: $first
                sort: FULL_PATH_ASC
            ) @connection(key: "VCSProviderWorkspaceList_workspaces") {
                totalCount
                edges {
                    node {
                        id
                        ...VCSProviderWorkspaceListItemFragment_workspace
                    }
                }
            }
        }
    `, queryData.node?.vcsProvider);

    const edges = data?.workspaces.edges ?? [];
    const totalCount = data?.workspaces.totalCount ?? 0;

    if (edges.length === 0) {
        return (
            <Typography sx={{ p: 4 }} align="center" color="textSecondary">
                No workspaces are linked to this VCS provider
            </Typography>
        );
    }

    return (
        <Box>
            <Typography sx={{ mb: 1, fontWeight: 500 }}>{totalCount} linked workspace{totalCount === 1 ? '' : 's'}</Typography>
            <InfiniteScroll
                dataLength={edges.length}
                next={() => loadNext(LOAD_MORE_COUNT)}
                hasMore={hasNext}
                loader={<ListSkeleton rowCount={3} />}
            >
                <List disablePadding>
                    {edges.map((edge, index) => edge?.node && (
                        <VCSProviderWorkspaceListItem
                            key={edge.node.id}
                            fragmentRef={edge.node}
                            last={index === edges.length - 1}
                        />
                    ))}
                </List>
            </InfiniteScroll>
        </Box>
    );
}

export default VCSProviderWorkspaceList;
