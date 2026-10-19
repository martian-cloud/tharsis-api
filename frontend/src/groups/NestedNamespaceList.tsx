import { Box, Button, List } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useLazyLoadQuery, usePaginationFragment } from 'react-relay/hooks';
import NamespaceListItem from './NamespaceListItem';
import { NestedNamespaceListFragment_namespaces$key } from './__generated__/NestedNamespaceListFragment_namespaces.graphql';
import { NestedNamespaceListPaginationQuery } from './__generated__/NestedNamespaceListPaginationQuery.graphql';
import { NestedNamespaceListQuery } from './__generated__/NestedNamespaceListQuery.graphql';

const INITIAL_ITEM_COUNT = 25;

interface Props {
    parentId: string
}

function NestedNamespaceList({ parentId }: Props) {
    const queryData = useLazyLoadQuery<NestedNamespaceListQuery>(graphql`
        query NestedNamespaceListQuery($first: Int, $last: Int, $after: String, $before: String, $parentId: String!) {
            ...NestedNamespaceListFragment_namespaces
        }
    `, { first: INITIAL_ITEM_COUNT, parentId });

    const { data, loadNext, hasNext, isLoadingNext } = usePaginationFragment<NestedNamespaceListPaginationQuery, NestedNamespaceListFragment_namespaces$key>(graphql`
        fragment NestedNamespaceListFragment_namespaces on Query
        @refetchable(queryName: "NestedNamespaceListPaginationQuery") {
            namespaces(
                after: $after
                before: $before
                first: $first
                last: $last
                parentId: $parentId
                sort: FULL_PATH_ASC
            ) @connection(key: "NestedNamespaceList_namespaces") {
                edges {
                    node {
                        id
                        ...NamespaceListItemFragment_namespace
                    }
                }
            }
        }
    `, queryData);

    const edges = data.namespaces?.edges ?? [];

    return (
        <List disablePadding>
            {edges.map((edge: any, index: number) => (
                <NamespaceListItem key={edge.node.id} namespaceKey={edge.node} last={!hasNext && index === edges.length - 1} />
            ))}
            {hasNext && (
                <Box sx={{ py: 1 }}>
                    <Button size="small" color="inherit" disabled={isLoadingNext} onClick={() => loadNext(INITIAL_ITEM_COUNT)}>
                        {isLoadingNext ? 'Loading...' : 'View more'}
                    </Button>
                </Box>
            )}
        </List>
    );
}

export default NestedNamespaceList;
