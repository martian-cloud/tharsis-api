import { Box, Button, List, Paper, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import throttle from 'lodash.throttle';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import InfiniteScroll from 'react-infinite-scroll-component';
import { useSnackbar } from 'notistack';
import { PreloadedQuery, fetchQuery, useFragment, useMutation, usePaginationFragment, usePreloadedQuery, useQueryLoader, useRelayEnvironment } from "react-relay/hooks";
import { Link as RouterLink, useSearchParams } from 'react-router-dom';
import ConfirmationDialog from '../../common/ConfirmationDialog';
import SearchInput from '../../common/SearchInput';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import ListSkeleton from '../../skeletons/ListSkeleton';
import GroupPackageListItem from './GroupPackageListItem';
import { GroupPackageListDeleteMutation } from './__generated__/GroupPackageListDeleteMutation.graphql';
import { GroupPackageListFragment_group$key } from './__generated__/GroupPackageListFragment_group.graphql';
import { GroupPackageListFragment_packages$key } from './__generated__/GroupPackageListFragment_packages.graphql';
import { GroupPackageListPaginationQuery } from './__generated__/GroupPackageListPaginationQuery.graphql';
import { GroupPackageListQuery } from './__generated__/GroupPackageListQuery.graphql';

const DESCRIPTION = 'Packages are versioned bundles used to store files such as OPA Rego policies';
const INITIAL_ITEM_COUNT = 20;

const query = graphql`
    query GroupPackageListQuery($first: Int, $last: Int, $after: String, $before: String, $groupId: String!, $search: String) {
        node(id: $groupId) {
            ...on Group {
                ...GroupPackageListFragment_packages
            }
        }
    }
`;

interface Group {
    id: string;
    fullPath: string;
}

interface InnerProps {
    group: Group;
    queryRef: PreloadedQuery<GroupPackageListQuery>;
    search?: string;
}

// packageToDelete is the package awaiting delete confirmation.
interface DeleteTarget {
    id: string;
    name: string;
}

function GroupPackageList({ group, queryRef, search = '' }: InnerProps) {
    const theme = useTheme();
    const [searchParams, setSearchParams] = useSearchParams();
    const [isRefreshing, setIsRefreshing] = useState(false);
    const { enqueueSnackbar } = useSnackbar();
    const [packageToDelete, setPackageToDelete] = useState<DeleteTarget | null>(null);

    const queryData = usePreloadedQuery<GroupPackageListQuery>(query, queryRef);

    const { data, loadNext, hasNext, refetch } = usePaginationFragment<GroupPackageListPaginationQuery, GroupPackageListFragment_packages$key>(
        graphql`
        fragment GroupPackageListFragment_packages on Group
        @refetchable(queryName: "GroupPackageListPaginationQuery") {
            # Count only, and deliberately unsearched: whether there is anything here to search is a
            # property of the group, not of what the current search matched.
            allPackages: packages(first: 0, includeInherited: true) {
                totalCount
            }
            packages(
                after: $after
                before: $before
                first: $first
                last: $last
                search: $search
                sort: GROUP_LEVEL_DESC
                includeInherited: true
            ) @connection(key: "PackageList_packages") {
                # The connection is filtered by $search, so its store id can't be reconstructed from
                # constants; __id hands back the id @deleteEdge needs for whatever search is live.
                __id
                edges {
                    node {
                        id
                        groupPath
                        ...GroupPackageListItemFragment_package
                    }
                }
            }
        }
        `, queryData.node
    );

    const edges = data?.packages?.edges || [];
    const connectionId = data?.packages?.__id;
    // Drives the choice between the list view and the "no packages" onboarding view. totalCount is
    // unfiltered, so a search matching nothing keeps the list view — and the search input with the
    // focus in it — mounted. It is fixed at load time though, so with no search on it is the edges
    // that tell us whether the group still has packages after one is deleted from the store.
    const hasPackages = (data?.allPackages?.totalCount ?? 0) > 0 && (edges.length > 0 || !!search);

    const environment = useRelayEnvironment();

    const fetch = useMemo(
        () =>
            throttle(
                (input: string | undefined, existingSearchParams: URLSearchParams) => {
                    setIsRefreshing(true);

                    fetchQuery(environment, query, {
                        first: INITIAL_ITEM_COUNT,
                        groupId: group.id,
                        search: input,
                    })
                        .subscribe({
                            complete: () => {
                                const nextParams = new URLSearchParams(existingSearchParams);
                                if (input?.trim()) {
                                    nextParams.set('search', input);
                                } else {
                                    nextParams.delete('search');
                                }
                                setSearchParams(nextParams, { replace: true });

                                refetch({
                                    first: INITIAL_ITEM_COUNT,
                                    search: input,
                                }, {
                                    fetchPolicy: 'store-only'
                                });

                                setIsRefreshing(false);
                            },
                            error: () => {
                                setIsRefreshing(false);
                            }
                        });
                },
                500,
                { leading: false, trailing: true }
            ),
        [environment, refetch, group.id],
    );

    const onSearchChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        fetch(event.target.value.toLowerCase(), searchParams);
    }, [fetch, searchParams]);

    const onKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
        if (event.key === 'Enter') {
            fetch.flush();
        }
    };

    const [commitDelete, deleteInFlight] = useMutation<GroupPackageListDeleteMutation>(graphql`
        mutation GroupPackageListDeleteMutation($input: DeletePackageInput!, $connections: [ID!]!) {
            deletePackage(input: $input) {
                package {
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

    const onDelete = (confirm?: boolean) => {
        if (!confirm || !packageToDelete) {
            setPackageToDelete(null);
            return;
        }

        commitDelete({
            variables: {
                input: { id: packageToDelete.id },
                // @deleteEdge drops the row from the list, so no re-read is needed.
                connections: connectionId ? [connectionId] : [],
            },
            onCompleted: response => {
                setPackageToDelete(null);
                if (response.deletePackage.problems.length) {
                    enqueueSnackbar(
                        response.deletePackage.problems.map(problem => problem.message).join('; '),
                        { variant: 'warning' }
                    );
                }
            },
            onError: error => {
                setPackageToDelete(null);
                enqueueSnackbar(`Unexpected error occurred: ${error.message}`, { variant: 'error' });
            }
        });
    };

    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={group.fullPath}
                childRoutes={[
                    { title: "packages", path: 'packages' }
                ]}
            />
            {hasPackages && <Box>
                <Box sx={{
                    display: 'flex',
                    flexDirection: 'row',
                    justifyContent: 'space-between',
                    [theme.breakpoints.down('md')]: {
                        flexDirection: 'column',
                        alignItems: 'flex-start',
                        '& > *': { marginBottom: 2 },
                    }
                }}>
                    <Box>
                        <Typography variant="h5" gutterBottom>Packages</Typography>
                        <Typography variant="body2">{DESCRIPTION}</Typography>
                    </Box>
                    {/* The wrapper keeps the button at its intrinsic height: as a direct child of this
                        row it would stretch to the full height of the heading block beside it. */}
                    <Box>
                        <Button sx={{ minWidth: 150 }} variant="outlined" component={RouterLink} to="new">New Package</Button>
                    </Box>
                </Box>
                <Box sx={{ display: 'flex', gap: 1, alignItems: 'flex-start', marginTop: 2, marginBottom: 2 }}>
                    <Box flex={1}>
                        <SearchInput
                            fullWidth
                            defaultValue={search}
                            placeholder="search for packages"
                            onChange={onSearchChange}
                            onKeyDown={onKeyDown}
                        />
                    </Box>
                </Box>
                <Paper sx={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0, border: `1px solid ${theme.palette.divider}` }}>
                    <Box padding={2} display="flex" alignItems="center" justifyContent="space-between">
                        <Typography variant="subtitle1">
                            {edges.length} package{edges.length === 1 ? '' : 's'}
                        </Typography>
                    </Box>
                </Paper>
                {(edges.length === 0) && !!search && <Typography
                    sx={{
                        padding: 4,
                        borderBottom: `1px solid ${theme.palette.divider}`,
                        borderLeft: `1px solid ${theme.palette.divider}`,
                        borderRight: `1px solid ${theme.palette.divider}`,
                        borderBottomLeftRadius: 4,
                        borderBottomRightRadius: 4
                    }}
                    align="center"
                    color="textSecondary"
                >
                    No packages matching search "{search}"
                </Typography>}
                <InfiniteScroll
                    dataLength={edges.length}
                    next={() => loadNext(20)}
                    hasMore={hasNext}
                    loader={<ListSkeleton rowCount={3} />}
                >
                    <List sx={isRefreshing ? { opacity: 0.5 } : null} disablePadding>
                        {edges.map((edge) => edge?.node && <GroupPackageListItem
                            key={edge.node.id}
                            fragmentRef={edge.node}
                            inherited={edge.node.groupPath !== group.fullPath}
                            onDelete={setPackageToDelete}
                        />)}
                    </List>
                </InfiniteScroll>
            </Box>}
            {packageToDelete && <ConfirmationDialog
                title="Delete Package"
                confirmLabel="Delete"
                confirmInProgress={deleteInFlight}
                onConfirm={() => onDelete(true)}
                onClose={() => onDelete()}
            >
                Are you sure you want to delete package <strong>{packageToDelete.name}</strong>? All of
                its versions will be deleted as well.
            </ConfirmationDialog>}
            {!hasPackages && <Box sx={{ marginTop: 4 }} display="flex" justifyContent="center">
                <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center" sx={{ maxWidth: 600 }}>
                    <Typography variant="h6">No packages found</Typography>
                    <Typography color="textSecondary" align="center" sx={{ marginBottom: 2 }}>
                        {DESCRIPTION}
                    </Typography>
                    <Button variant="outlined" component={RouterLink} to="new">New Package</Button>
                </Box>
            </Box>}
        </Box>
    );
}

interface ContainerProps {
    fragmentRef: GroupPackageListFragment_group$key
}

function GroupPackageListContainer({ fragmentRef }: ContainerProps) {
    const group = useFragment<GroupPackageListFragment_group$key>(
        graphql`
        fragment GroupPackageListFragment_group on Group {
            id
            fullPath
        }
        `, fragmentRef);

    const [queryRef, loadQuery] = useQueryLoader<GroupPackageListQuery>(query);
    const [searchParams] = useSearchParams();

    const search = searchParams.get('search') || undefined;

    useEffect(() => {
        loadQuery({
            first: INITIAL_ITEM_COUNT,
            groupId: group.id,
            search,
        }, { fetchPolicy: 'store-and-network' });
    }, [loadQuery, search, group.id]);

    return queryRef != null ? (
        <GroupPackageList
            group={group}
            queryRef={queryRef}
            search={search}
        />
    ) : null;
}

export default GroupPackageListContainer;
