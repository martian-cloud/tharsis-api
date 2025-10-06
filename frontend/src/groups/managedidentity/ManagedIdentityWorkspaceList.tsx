import { Box, List, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import InfiniteScroll from 'react-infinite-scroll-component';
import { useLazyLoadQuery, usePaginationFragment } from "react-relay/hooks";
import ListSkeleton from '../../skeletons/ListSkeleton';
import { ManagedIdentityWorkspaceListFragment_workspaces$key } from './__generated__/ManagedIdentityWorkspaceListFragment_workspaces.graphql';
import { ManagedIdentityWorkspaceListPaginationQuery } from './__generated__/ManagedIdentityWorkspaceListPaginationQuery.graphql';
import { ManagedIdentityWorkspaceListQuery } from './__generated__/ManagedIdentityWorkspaceListQuery.graphql';
import ManagedIdentityWorkspaceListItem from './ManagedIdentityWorkspaceListItem';
import React from 'react';

const INITIAL_ITEM_COUNT = 50;

interface Props {
    managedIdentityId: string
}

const query = graphql`
    query ManagedIdentityWorkspaceListQuery($first: Int, $after: String, $managedIdentityId: String!) {
        node(id: $managedIdentityId) {
            ...on ManagedIdentity {
                ...ManagedIdentityWorkspaceListFragment_workspaces
            }
        }
    }
`;

function ManagedIdentityWorkspaceList({ managedIdentityId }: Props) {
    const queryData = useLazyLoadQuery<ManagedIdentityWorkspaceListQuery>(query, { first: INITIAL_ITEM_COUNT, managedIdentityId }, { fetchPolicy: 'store-and-network' });

    const { data, loadNext, hasNext } = usePaginationFragment<ManagedIdentityWorkspaceListPaginationQuery, ManagedIdentityWorkspaceListFragment_workspaces$key>(
        graphql`
        fragment ManagedIdentityWorkspaceListFragment_workspaces on ManagedIdentity
        @refetchable(queryName: "ManagedIdentityWorkspaceListPaginationQuery") {
            workspaces(
                after: $after
                first: $first
                sort:FULL_PATH_ASC
            ) @connection(key: "ManagedIdentityWorkspaceList_workspaces") {
                totalCount
                edges {
                    node {
                        id
                        ...ManagedIdentityWorkspaceListItemFragment_workspace
                    }
                }
            }
        }
    `, queryData.node);

    const edgeCount = (data?.workspaces.edges?.length ?? 0);

    return (
        <Box>
            {edgeCount === 0 && <Typography
                sx={{ p: 4 }}
                align="center"
                color="textSecondary"
            >
                No workspaces assigned
            </Typography>}
            {edgeCount > 0 && <React.Fragment>
                <Typography mb={1} fontWeight={500}>{data?.workspaces.totalCount} assigned workspace{data?.workspaces.totalCount === 1 ? '' : 's'}</Typography>
                <InfiniteScroll
                    dataLength={data?.workspaces.edges?.length ?? 0}
                    next={() => loadNext(INITIAL_ITEM_COUNT)}
                    hasMore={hasNext}
                    loader={<ListSkeleton rowCount={3} />}
                >
                    <List disablePadding>
                        {data?.workspaces.edges?.map((edge: any, index: number) => <ManagedIdentityWorkspaceListItem key={edge.node.id} workspaceKey={edge.node} last={index === (edgeCount - 1)} />)}
                    </List>
                </InfiniteScroll>
            </React.Fragment>}
        </Box>
    );
}

export default ManagedIdentityWorkspaceList;
