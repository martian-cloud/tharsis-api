import FilterListIcon from '@mui/icons-material/FilterList';
import Badge from '@mui/material/Badge';
import Box from '@mui/material/Box';
import IconButton from '@mui/material/IconButton';
import List from '@mui/material/List';
import Paper from '@mui/material/Paper';
import { useTheme } from '@mui/material/styles';
import Typography from '@mui/material/Typography';
import graphql from 'babel-plugin-relay/macro';
import throttle from 'lodash.throttle';
import React, { useMemo, useState } from 'react';
import InfiniteScroll from 'react-infinite-scroll-component';
import { fetchQuery, PreloadedQuery, usePaginationFragment, usePreloadedQuery, useRelayEnvironment } from 'react-relay/hooks';
import { useSearchParams } from 'react-router-dom';
import SearchInput from '../common/SearchInput';
import ListSkeleton from '../skeletons/ListSkeleton';
import { WorkspaceSearchFragment_workspaces$key } from './__generated__/WorkspaceSearchFragment_workspaces.graphql';
import { WorkspaceSearchPaginationQuery } from './__generated__/WorkspaceSearchPaginationQuery.graphql';
import { WorkspaceSearchQuery } from './__generated__/WorkspaceSearchQuery.graphql';
import type { LabelFilter as LabelFilterType } from './labels/LabelFilter';
import LabelFilter from './labels/LabelFilter';
import WorkspaceSearchListItem from './WorkspaceSearchListItem';

export const INITIAL_ITEM_COUNT = 100;

const query = graphql`
    query WorkspaceSearchQuery($first: Int, $last: Int, $after: String, $before: String, $search: String, $labelFilter: WorkspaceLabelsFilter) {
      ...WorkspaceSearchFragment_workspaces
    }
`;

interface Props {
  queryRef: PreloadedQuery<WorkspaceSearchQuery>
  search?: string
  labelFilters?: LabelFilterType[]
  filterExpanded?: boolean
}

function WorkspaceSearch({ search = '', labelFilters = [], filterExpanded = false, queryRef }: Props) {
  const queryData = usePreloadedQuery<WorkspaceSearchQuery>(query, queryRef);
  const [searchParams, setSearchParams] = useSearchParams();
  const theme = useTheme();
  const environment = useRelayEnvironment();

  const [isRefreshing, setIsRefreshing] = useState(false);

  const { data, loadNext, hasNext, refetch } = usePaginationFragment<WorkspaceSearchPaginationQuery, WorkspaceSearchFragment_workspaces$key>(
    graphql`
    fragment WorkspaceSearchFragment_workspaces on Query
    @refetchable(queryName: "WorkspaceSearchPaginationQuery") {
      workspaces(
          after: $after
          before: $before
          first: $first
          last: $last
          search: $search
          labelFilter: $labelFilter
          sort:FULL_PATH_ASC
      ) @connection(key: "WorkspaceSearch_workspaces") {
          totalCount
          edges {
              node {
                  id
                  ...WorkspaceSearchListItemFragment_workspace
              }
          }
      }
    }
  `, queryData);

  const fetch = useMemo(
    () =>
      throttle(
        (input: string, filters: LabelFilterType[], existingSearchParams: URLSearchParams) => {
          setIsRefreshing(true);

          fetchQuery(environment, query, {
            first: INITIAL_ITEM_COUNT,
            search: input,
            labelFilter: {
              labels: filters
            }
          })
            .subscribe({
              complete: () => {
                if (input.trim() !== '') {
                  existingSearchParams.set('search', input);
                } else {
                  existingSearchParams.delete('search');
                }

                // Create set of label keys for fast lookup
                const filterKeys = new Set(filters.map(f => f.key));

                // Remove filters that are no longer present
                existingSearchParams.forEach((_, key) => {
                  if (key.startsWith('label.') && !filterKeys.has(key.substring(6))) {
                    existingSearchParams.delete(key);
                  }
                });
                // Add new filters
                filters.forEach(filter => {
                  existingSearchParams.set(`label.${filter.key}`, filter.value);
                });

                setSearchParams(existingSearchParams, { replace: true });

                // *After* the query has been fetched, we call
                // refetch again to re-render with the updated data.
                // At this point the data for the query should
                // be cached, so we use the 'store-only'
                // fetchPolicy to avoid suspending.
                refetch({
                  first: INITIAL_ITEM_COUNT,
                  search: input,
                  labelFilter: {
                    labels: filters
                  }
                }, { fetchPolicy: 'store-only' });

                setIsRefreshing(false);
              },
              error: () => {
                setIsRefreshing(false);
              }
            });
        },
        2000,
        { leading: false, trailing: true }
      ),
    [environment, refetch],
  );

  const onKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    // Only handle enter key
    if (event.key === 'Enter') {
      fetch.flush();
    }
  };

  const onSearchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const newSearch = event.target.value.toLowerCase();
    fetch(newSearch, labelFilters, searchParams);
  };

  const onLabelFiltersChange = (newFilters: LabelFilterType[]) => {
    fetch(search, newFilters, searchParams);
    fetch.flush();
  };

  const onFilterExpandedChange = (expanded: boolean) => {
    expanded ? searchParams.set('filterExpanded', 'true') : searchParams.delete('filterExpanded');
    setSearchParams(searchParams, { replace: true });
  };

  return (
    <Box maxWidth={1200} margin="auto" padding={2}>

      {(search !== '' || labelFilters.length > 0 || data.workspaces?.edges?.length !== 0) && <React.Fragment>
        <Typography variant="h5" sx={{ marginBottom: 2 }}>Workspaces</Typography>
        <Box marginBottom={2} display="flex" gap={1} alignItems="flex-start">
          <Box flex={1}>
            <SearchInput
              fullWidth
              defaultValue={search}
              placeholder="search for workspaces"
              onChange={onSearchChange}
              onKeyDown={onKeyDown}
            />
          </Box>
          <Box>
            <IconButton
              onClick={() => onFilterExpandedChange(!filterExpanded)}
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
        <Paper sx={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0, border: `1px solid ${theme.palette.divider}` }}>
          <Box padding={2} display="flex" alignItems="center" justifyContent="space-between">
            <Typography variant="subtitle1">
              {data.workspaces.totalCount} workspace{data.workspaces.totalCount === 1 ? '' : 's'}
            </Typography>
          </Box>
        </Paper>
        {(!data.workspaces.edges || data.workspaces.edges?.length === 0) && (search !== '' || labelFilters.length > 0) && <Typography
          align="center"
          color="textSecondary"
          sx={{
            padding: 4,
            borderBottom: `1px solid ${theme.palette.divider}`,
            borderLeft: `1px solid ${theme.palette.divider}`,
            borderRight: `1px solid ${theme.palette.divider}`,
            borderBottomLeftRadius: 4,
            borderBottomRightRadius: 4
          }}
        >
          No workspaces matching {search && labelFilters.length > 0 ? 'search and filters' : search ? `search "${search}"` : 'filters'}
        </Typography>}
        <InfiniteScroll
          dataLength={data.workspaces.edges?.length ?? 0}
          next={() => loadNext(INITIAL_ITEM_COUNT)}
          hasMore={hasNext}
          loader={<ListSkeleton rowCount={3} />}
        >
          <List
            disablePadding
            sx={{
              opacity: isRefreshing ? 0.5 : 1,
              transition: 'opacity 0.3s ease-in-out'
            }}
          >
            {data.workspaces.edges?.map((edge: any) => <WorkspaceSearchListItem
              key={edge.node.id}
              workspaceKey={edge.node}
            />)}
          </List>
        </InfiniteScroll>
      </React.Fragment>}

      {!search && labelFilters.length === 0 && data.workspaces.edges?.length === 0 && <Box sx={{ marginTop: 4 }} display="flex" justifyContent="center">
        <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center" sx={{ maxWidth: 600 }}>
          <Typography variant="h6">You don't have access to any workspaces</Typography>
        </Box>
      </Box>}
    </Box>
  );
}

export default WorkspaceSearch;
