import React, { useEffect, useMemo, useState } from 'react';
import TreeViewIcon from '@mui/icons-material/AccountTree';
import ListViewIcon from '@mui/icons-material/ViewList';
import { Box, List, ToggleButton, ToggleButtonGroup, Tooltip, Typography } from '@mui/material';
import throttle from 'lodash.throttle';
import graphql from 'babel-plugin-relay/macro';
import InfiniteScroll from 'react-infinite-scroll-component';
import { fetchQuery, useLazyLoadQuery, usePaginationFragment, useRelayEnvironment } from 'react-relay/hooks';
import ListSkeleton from '../skeletons/ListSkeleton';
import NamespaceListItem from './NamespaceListItem';
import SearchInput from '../common/SearchInput';
import { NamespaceListFragment_namespaces$key } from './__generated__/NamespaceListFragment_namespaces.graphql';
import { NamespaceListPaginationQuery } from './__generated__/NamespaceListPaginationQuery.graphql';
import { NamespaceListQuery } from './__generated__/NamespaceListQuery.graphql';

const INITIAL_ITEM_COUNT = 50;

// Tree lists one level at a time and expands groups in place; list shows every level below as a flat list
type View = 'tree' | 'list';

// A search always spans every level, so its results are flat whichever view is selected
function queryVariables(parentPath: string | undefined, view: View, search?: string) {
    const flat = view === 'list' || !!search;
    // parentId accepts a TRN, so the group doesn't need to be looked up by ID first
    const parentId = parentPath ? `trn:group:${parentPath}` : undefined;
    return { first: INITIAL_ITEM_COUNT, parentId, search, includeNested: flat, topLevel: !parentId && !flat };
}

interface Props {
    // Full path of the group whose direct children are listed; search results show their path relative to it.
    // Omit to list the top of the caller's hierarchy
    parentPath?: string
    // Shown above the search and view controls
    header?: React.ReactNode
    // Shown instead of the default message when there is nothing to list. When given, an empty list
    // with no search active shows only this, without the header or the search and view controls
    emptyState?: React.ReactNode
}

const query = graphql`
    query NamespaceListQuery($first: Int, $last: Int, $after: String, $before: String, $parentId: String, $search: String, $topLevel: Boolean, $includeNested: Boolean) {
        ...NamespaceListFragment_namespaces
    }
`;

function NamespaceList({ parentPath, header, emptyState }: Props) {
    const [search, setSearch] = useState<string | undefined>('');
    const [isRefreshing, setIsRefreshing] = useState(false);
    const [view, setView] = useState<View>('tree');
    const [isSwitchingView, setIsSwitchingView] = useState(false);

    // Only the first load goes through here; later view and search changes use fetchQuery + refetch below.
    // Passing the current view would change the variables on every switch and suspend the whole list.
    const [initialVariables] = useState(() => queryVariables(parentPath, view));

    const queryData = useLazyLoadQuery<NamespaceListQuery>(
        query,
        initialVariables,
        { fetchPolicy: 'store-and-network' }
    );

    const { data, loadNext, hasNext, refetch } = usePaginationFragment<NamespaceListPaginationQuery, NamespaceListFragment_namespaces$key>(
        graphql`
            fragment NamespaceListFragment_namespaces on Query
            @refetchable(queryName: "NamespaceListPaginationQuery") {
                namespaces(
                    after: $after
                    before: $before
                    first: $first
                    includeNested: $includeNested
                    last: $last
                    parentId: $parentId
                    search: $search
                    sort: FULL_PATH_ASC
                    topLevel: $topLevel
                ) @connection(key: "NamespaceList_namespaces") {
                    edges {
                        node {
                            id
                            ...NamespaceListItemFragment_namespace
                        }
                    }
                }
            }
        `, queryData);

    const environment = useRelayEnvironment();

    const load = useMemo(
        () => (input: string | undefined, nextView: View, onDone?: () => void) => {
            setIsRefreshing(true);

            const variables = queryVariables(parentPath, nextView, input?.trim());

            fetchQuery(environment, query, variables)
                .subscribe({
                    complete: () => {
                        setIsRefreshing(false);
                        setSearch(input);
                        refetch(variables, { fetchPolicy: 'store-only' });
                        onDone?.();
                    },
                    error: () => {
                        setIsRefreshing(false);
                        onDone?.();
                    }
                });
        },
        [environment, refetch, parentPath],
    );

    const fetch = useMemo(
        () => throttle((input?: string) => load(input, view), 2000, { leading: false, trailing: true }),
        [load, view],
    );

    useEffect(() => {
        return () => {
            fetch.cancel();
        };
    }, [fetch]);

    const onSearchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        fetch(event.target.value.toLowerCase());
    };

    const onKeyPress = (event: React.KeyboardEvent<HTMLInputElement>) => {
        if (event.which === 13) {
            fetch.flush();
        }
    };

    const onViewChange = (_event: React.MouseEvent<HTMLElement>, value: View | null) => {
        // ToggleButtonGroup reports null when the active button is clicked again; keep the current view
        if (value && value !== view) {
            setView(value);
            setIsSwitchingView(true);
            load(search, value, () => setIsSwitchingView(false));
        }
    };

    const edges = data.namespaces?.edges ?? [];
    const edgeCount = edges.length - 1;
    const flat = view === 'list' || !!search;

    if (emptyState && edges.length === 0 && !search && !isRefreshing && !isSwitchingView) {
        return <React.Fragment>{emptyState}</React.Fragment>;
    }

    return (
        <React.Fragment>
            {header}
            <Box display="flex" alignItems="stretch" gap={1} sx={{ marginBottom: 2 }}>
                <SearchInput
                    placeholder="search namespaces"
                    fullWidth
                    onChange={onSearchChange}
                    onKeyPress={onKeyPress}
                />
                <ToggleButtonGroup size="small" value={view} exclusive onChange={onViewChange} sx={{ flexShrink: 0 }}>
                    <ToggleButton value="tree" aria-label="tree view">
                        <Tooltip title="Tree view"><TreeViewIcon fontSize="small" /></Tooltip>
                    </ToggleButton>
                    <ToggleButton value="list" aria-label="list view">
                        <Tooltip title="List all nested namespaces"><ListViewIcon fontSize="small" /></Tooltip>
                    </ToggleButton>
                </ToggleButtonGroup>
            </Box>
            {isSwitchingView ? <ListSkeleton rowCount={10} /> : (
                <React.Fragment>
                    {edges.length === 0 && search !== '' && (
                        <Typography sx={{ p: 4 }} align="center" color="textSecondary">
                            No namespaces matching search <strong>{search}</strong>
                        </Typography>
                    )}
                    {edges.length === 0 && !search && (emptyState ?? (
                        <Box padding={4} display="flex" justifyContent="center" alignItems="center">
                            <Typography color="textSecondary">No subgroups or workspaces in this group</Typography>
                        </Box>
                    ))}
                    <InfiniteScroll
                        dataLength={edges.length}
                        next={() => loadNext(20)}
                        hasMore={hasNext}
                        loader={<ListSkeleton rowCount={3} />}
                    >
                        <List sx={isRefreshing ? { opacity: 0.5 } : null} disablePadding>
                            {edges.map((edge: any, index: number) => (
                                <NamespaceListItem
                                    key={edge.node.id}
                                    namespaceKey={edge.node}
                                    last={index === edgeCount}
                                    // A flat list already includes every level, so rows show their path and don't expand
                                    expandable={!flat}
                                    // The top level can include namespaces nested in groups the caller isn't a member
                                    // of, so it shows full paths; a root group's full path is just its name
                                    pathRelativeTo={flat || !parentPath ? (parentPath ?? '') : undefined}
                                />
                            ))}
                        </List>
                    </InfiniteScroll>
                </React.Fragment>
            )}
        </React.Fragment>
    );
}

export default NamespaceList;
