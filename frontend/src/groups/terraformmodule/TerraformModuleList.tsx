import FilterListIcon from '@mui/icons-material/FilterList';
import { Badge, Box, IconButton, List, Paper, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import throttle from 'lodash.throttle';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import InfiniteScroll from 'react-infinite-scroll-component';
import { PreloadedQuery, fetchQuery, useFragment, usePaginationFragment, usePreloadedQuery, useQueryLoader, useRelayEnvironment } from "react-relay/hooks";
import { useSearchParams } from 'react-router-dom';
import SearchInput from '../../common/SearchInput';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import ListSkeleton from '../../skeletons/ListSkeleton';
import LabelFilter, { LabelFilterItem } from '../../workspace/labels/LabelFilter';
import TerraformModuleListItem from './TerraformModuleListItem';
import { TerraformModuleListFragment_group$key } from './__generated__/TerraformModuleListFragment_group.graphql';
import { TerraformModuleListFragment_terraformModules$key } from './__generated__/TerraformModuleListFragment_terraformModules.graphql';
import { TerraformModuleListPaginationQuery } from './__generated__/TerraformModuleListPaginationQuery.graphql';
import { TerraformModuleListQuery } from './__generated__/TerraformModuleListQuery.graphql';

const DESCRIPTION = 'Terraform modules provide reusable infrastructure components that can be shared across workspaces';
const INITIAL_ITEM_COUNT = 20;

const query = graphql`
    query TerraformModuleListQuery($first: Int, $last: Int, $after: String, $before: String, $groupId: String!, $search: String, $labelFilter: TerraformModuleLabelsFilter) {
        node(id: $groupId) {
            ...on Group {
                ...TerraformModuleListFragment_terraformModules
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
    queryRef: PreloadedQuery<TerraformModuleListQuery>;
    search?: string;
    labelFilters?: LabelFilterItem[];
    filterExpanded?: boolean;
}

function TerraformModuleList({ group, queryRef, search = '', labelFilters = [], filterExpanded = false }: InnerProps) {
    const theme = useTheme();
    const [searchParams, setSearchParams] = useSearchParams();
    const [isRefreshing, setIsRefreshing] = useState(false);

    const queryData = usePreloadedQuery<TerraformModuleListQuery>(query, queryRef);

    const { data, loadNext, hasNext, refetch } = usePaginationFragment<TerraformModuleListPaginationQuery, TerraformModuleListFragment_terraformModules$key>(
        graphql`
        fragment TerraformModuleListFragment_terraformModules on Group
        @refetchable(queryName: "TerraformModuleListPaginationQuery") {
            terraformModules(
                after: $after
                before: $before
                first: $first
                last: $last
                search: $search
                sort: GROUP_LEVEL_DESC
                includeInherited: true
                labelFilter: $labelFilter
            ) @connection(key: "TerraformModuleList_terraformModules") {
                totalCount
                edges {
                    node {
                        id
                        groupPath
                        ...TerraformModuleListItemFragment_terraformModule
                    }
                }
            }
        }
        `, queryData.node
    );

    const edges = data?.terraformModules?.edges || [];

    const environment = useRelayEnvironment();

    const fetch = useMemo(
        () =>
            throttle(
                (input: string | undefined, filters: LabelFilterItem[], existingSearchParams: URLSearchParams) => {
                    setIsRefreshing(true);

                    fetchQuery(environment, query, {
                        first: INITIAL_ITEM_COUNT,
                        groupId: group.id,
                        search: input,
                        labelFilter: { labels: filters },
                    })
                        .subscribe({
                            complete: () => {
                                const nextParams = new URLSearchParams(existingSearchParams);
                                if (input?.trim()) {
                                    nextParams.set('search', input);
                                } else {
                                    nextParams.delete('search');
                                }

                                const filterKeys = new Set(filters.map(f => f.key));
                                nextParams.forEach((_, key) => {
                                    if (key.startsWith('label.') && !filterKeys.has(key.substring(6))) {
                                        nextParams.delete(key);
                                    }
                                });
                                filters.forEach(filter => {
                                    nextParams.set(`label.${filter.key}`, filter.value);
                                });

                                setSearchParams(nextParams, { replace: true });

                                refetch({
                                    first: INITIAL_ITEM_COUNT,
                                    search: input,
                                    labelFilter: { labels: filters },
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
        fetch(event.target.value.toLowerCase(), labelFilters, searchParams);
    }, [fetch, labelFilters, searchParams]);

    const onKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
        if (event.key === 'Enter') {
            fetch.flush();
        }
    };

    const onLabelFiltersChange = useCallback((newFilters: LabelFilterItem[]) => {
        fetch(search, newFilters, searchParams);
        fetch.flush();
    }, [fetch, search, searchParams]);

    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={group.fullPath}
                childRoutes={[
                    { title: "terraform modules", path: 'terraform_modules' }
                ]}
            />
            {(!!search || labelFilters.length > 0 || edges.length !== 0) && <Box>
                <Box>
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
                            <Typography variant="h5" gutterBottom>Terraform Modules</Typography>
                            <Typography variant="body2">
                                {DESCRIPTION}
                            </Typography>
                        </Box>
                    </Box>
                    <Box sx={{ display: 'flex', gap: 1, alignItems: 'flex-start', marginTop: 2, marginBottom: 2 }}>
                        <Box flex={1}>
                            <SearchInput
                                fullWidth
                                defaultValue={search}
                                placeholder="search for terraform modules"
                                onChange={onSearchChange}
                                onKeyDown={onKeyDown}
                            />
                        </Box>
                        <Box>
                            <IconButton
                                onClick={() => {
                                    const next = !filterExpanded;
                                    const nextParams = new URLSearchParams(searchParams);
                                    if (next) {
                                        nextParams.set('filterExpanded', 'true');
                                    } else {
                                        nextParams.delete('filterExpanded');
                                    }
                                    setSearchParams(nextParams, { replace: true });
                                }}
                                color={filterExpanded || labelFilters.length > 0 ? 'primary' : 'default'}
                                sx={{
                                    border: `1px solid ${theme.palette.divider}`,
                                    borderRadius: 1,
                                    height: 40,
                                    width: 40
                                }}
                                aria-label="Toggle label filters"
                            >
                                <Badge badgeContent={labelFilters.length} color="primary">
                                    <FilterListIcon />
                                </Badge>
                            </IconButton>
                        </Box>
                    </Box>
                    {filterExpanded && (
                        <Box marginBottom={2}>
                            <LabelFilter
                                filters={labelFilters}
                                onFiltersChange={onLabelFiltersChange}
                                expanded={filterExpanded}
                            />
                        </Box>
                    )}
                </Box>
                <Paper sx={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0, border: `1px solid ${theme.palette.divider}` }}>
                    <Box padding={2} display="flex" alignItems="center" justifyContent="space-between">
                        <Typography variant="subtitle1">
                            {edges.length} terraform module{edges.length === 1 ? '' : 's'}
                        </Typography>
                    </Box>
                </Paper>
                {(edges.length === 0) && (!!search || labelFilters.length > 0) && <Typography
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
                    No terraform modules matching {search && labelFilters.length > 0 ? 'search and filters' : search ? `search "${search}"` : 'filters'}
                </Typography>}
                <InfiniteScroll
                    dataLength={edges.length}
                    next={() => loadNext(20)}
                    hasMore={hasNext}
                    loader={<ListSkeleton rowCount={3} />}
                >
                    <List sx={isRefreshing ? { opacity: 0.5 } : null} disablePadding>
                        {edges.map((edge) => edge?.node && <TerraformModuleListItem
                            key={edge.node.id}
                            fragmentRef={edge.node}
                            inherited={edge.node.groupPath !== group.fullPath}
                        />)}
                    </List>
                </InfiniteScroll>
            </Box>}
            {!search && labelFilters.length === 0 && edges.length === 0 && <Box sx={{ marginTop: 4 }} display="flex" justifyContent="center">
                <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center" sx={{ maxWidth: 600 }}>
                    <Typography variant="h6">No terraform modules found</Typography>
                    <Typography color="textSecondary" align="center" sx={{ marginBottom: 2 }}>
                        {DESCRIPTION}
                    </Typography>
                </Box>
            </Box>}
        </Box>
    );
}

interface ContainerProps {
    fragmentRef: TerraformModuleListFragment_group$key
}

function TerraformModuleListContainer({ fragmentRef }: ContainerProps) {
    const group = useFragment<TerraformModuleListFragment_group$key>(
        graphql`
        fragment TerraformModuleListFragment_group on Group {
            id
            fullPath
        }
        `, fragmentRef);

    const [queryRef, loadQuery] = useQueryLoader<TerraformModuleListQuery>(query);
    const [searchParams] = useSearchParams();

    const search = searchParams.get('search') || undefined;
    const filterExpanded = searchParams.get('filterExpanded') === 'true';
    const labelFilters = useMemo<LabelFilterItem[]>(() => {
        const filters: LabelFilterItem[] = [];
        searchParams.forEach((value, key) => {
            if (key.startsWith('label.')) filters.push({ key: key.substring(6), value });
        });
        return filters;
    }, [searchParams]);

    useEffect(() => {
        loadQuery({
            first: INITIAL_ITEM_COUNT,
            groupId: group.id,
            search,
            labelFilter: { labels: labelFilters },
        }, { fetchPolicy: 'store-and-network' });
    }, [loadQuery]);

    return queryRef != null ? (
        <TerraformModuleList
            group={group}
            queryRef={queryRef}
            search={search}
            labelFilters={labelFilters}
            filterExpanded={filterExpanded}
        />
    ) : null;
}

export default TerraformModuleListContainer;
