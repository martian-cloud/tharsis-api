import { Box, List, Paper, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import InfiniteScroll from 'react-infinite-scroll-component';
import ListSkeleton from '../../../skeletons/ListSkeleton';
import ManagedIdentityAliasesListItem from './ManagedIdentityAliasesListItem';
import { ConnectionHandler, usePaginationFragment } from 'react-relay/hooks';
import { ManagedIdentityAliasesListFragment_managedIdentity$key } from './__generated__/ManagedIdentityAliasesListFragment_managedIdentity.graphql';
import { ManagedIdentityAliasesListPaginationQuery } from './__generated__/ManagedIdentityAliasesListPaginationQuery.graphql';

export const INITIAL_ITEM_COUNT = 50;

export function GetConnections(managedIdentityId: string): [string] {
    const connectionId = ConnectionHandler.getConnectionID(
        managedIdentityId,
        'ManagedIdentityAliasesList_aliases',
    );
    return [connectionId];
}

interface Props {
    fragmentRef: ManagedIdentityAliasesListFragment_managedIdentity$key
}

function ManagedIdentityAliasesList({ fragmentRef }: Props) {
    const theme = useTheme();

    const { data, loadNext, hasNext } = usePaginationFragment<ManagedIdentityAliasesListPaginationQuery, ManagedIdentityAliasesListFragment_managedIdentity$key>(
        graphql`
            fragment ManagedIdentityAliasesListFragment_managedIdentity on ManagedIdentity
            @refetchable(queryName: "ManagedIdentityAliasesListPaginationQuery")
             {
                id
                aliases(
                    first: $first
                    last: $last
                    after: $after
                    before: $before
                ) @connection(key: "ManagedIdentityAliasesList_aliases") {
                    edges {
                        node {
                            id
                            ...ManagedIdentityAliasesListItemFragment_managedIdentity
                        }
                    }
                }
            }`, fragmentRef);

    return (data.aliases.edges && data.aliases.edges?.length > 0) ?
        <Box>
            <Paper sx={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0, border: `1px solid ${theme.palette.divider}` }}>
                <Box padding={2} display="flex" alignItems="center" justifyContent="space-between">
                    <Typography variant="subtitle1">
                    {data.aliases.edges.length} alias{data.aliases.edges.length === 1 ? '' : 'es'}</Typography>
                </Box>
            </Paper>
            <InfiniteScroll
                dataLength={data.aliases.edges.length ?? 0}
                next={() => loadNext(20)}
                hasMore={hasNext}
                loader={<ListSkeleton rowCount={3} />}
            >
                <List
                    disablePadding
                > {data.aliases.edges.map((edge: any) => <ManagedIdentityAliasesListItem
                    key={edge.node.id}
                    fragmentRef={edge.node}
                    />)}
                </List>
            </InfiniteScroll>
        </Box>
        :
        <Paper sx={{ p: 2 }}>
            <Typography>No aliases exist for this managed identity</Typography>
        </Paper>
}

export default ManagedIdentityAliasesList
