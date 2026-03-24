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
import React, { useCallback, useMemo, useState } from 'react';
import InfiniteScroll from 'react-infinite-scroll-component';
import { fetchQuery, PreloadedQuery, usePaginationFragment, usePreloadedQuery, useRelayEnvironment } from 'react-relay/hooks';
import { useSearchParams } from 'react-router-dom';
import SearchInput from '../common/SearchInput';
import LabelFilter, { LabelFilterItem } from '../workspace/labels/LabelFilter';
import ListSkeleton from '../skeletons/ListSkeleton';
import TerraformModuleSearchListItem from './TerraformModuleSearchListItem';
import { TerraformModuleSearchFragment_modules$key } from './__generated__/TerraformModuleSearchFragment_modules.graphql';
import { TerraformModuleSearchPaginationQuery } from './__generated__/TerraformModuleSearchPaginationQuery.graphql';
import { TerraformModuleSearchQuery } from './__generated__/TerraformModuleSearchQuery.graphql';

export const INITIAL_ITEM_COUNT = 50;

const query = graphql`
    query TerraformModuleSearchQuery($first: Int, $last: Int, $after: String, $before: String, $search: String, $labelFilter: TerraformModuleLabelsFilter) {
      ...TerraformModuleSearchFragment_modules
    }
`;

interface Props {
  queryRef: PreloadedQuery<TerraformModuleSearchQuery>
  search?: string
  labelFilters?: LabelFilterItem[]
  filterExpanded?: boolean
}

function TerraformModuleSearch({ search = '', labelFilters = [], filterExpanded = false, queryRef }: Props) {
  const queryData = usePreloadedQuery<TerraformModuleSearchQuery>(query, queryRef);
  const [searchParams, setSearchParams] = useSearchParams();
  const theme = useTheme();
  const environment = useRelayEnvironment();

  const [isRefreshing, setIsRefreshing] = useState(false);

  const { data, loadNext, hasNext, refetch } = usePaginationFragment<TerraformModuleSearchPaginationQuery, TerraformModuleSearchFragment_modules$key>(
    graphql`
    fragment TerraformModuleSearchFragment_modules on Query
    @refetchable(queryName: "TerraformModuleSearchPaginationQuery") {
      terraformModules(
          after: $after
          before: $before
          first: $first
          last: $last
          search: $search
          sort: NAME_ASC
          labelFilter: $labelFilter
      ) @connection(key: "TerraformModuleSearch_terraformModules") {
          totalCount
          edges {
              node {
                  id
                  ...TerraformModuleSearchListItemFragment_module
              }
          }
      }
    }
  `, queryData);

  const fetch = useMemo(
    () =>
      throttle(
        (input: string, filters: LabelFilterItem[], existingSearchParams: URLSearchParams) => {
          setIsRefreshing(true);

          fetchQuery(environment, query, {
            first: INITIAL_ITEM_COUNT,
            search: input,
            labelFilter: { labels: filters }
          })
            .subscribe({
              complete: () => {
                const nextParams = new URLSearchParams(existingSearchParams);
                if (input.trim() !== '') {
                  nextParams.set('search', input);
                } else {
                  nextParams.delete('search');
                }

                // Create set of label keys for fast lookup
                const filterKeys = new Set(filters.map(f => f.key));

                // Remove filters that are no longer present
                nextParams.forEach((_, key) => {
                  if (key.startsWith('label.') && !filterKeys.has(key.substring(6))) {
                    nextParams.delete(key);
                  }
                });
                // Add new filters
                filters.forEach(filter => {
                  nextParams.set(`label.${filter.key}`, filter.value);
                });

                setSearchParams(nextParams, { replace: true });

                // *After* the query has been fetched, we call
                // refetch again to re-render with the updated data.
                // At this point the data for the query should
                // be cached, so we use the 'store-only'
                // fetchPolicy to avoid suspending.
                refetch({
                  first: INITIAL_ITEM_COUNT,
                  search: input,
                  labelFilter: { labels: filters }
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

  const onSearchChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    const newSearch = event.target.value.toLowerCase();
    fetch(newSearch, labelFilters, searchParams);
  }, [fetch, labelFilters, searchParams]);

  const onLabelFiltersChange = useCallback((newFilters: LabelFilterItem[]) => {
    fetch(search, newFilters, searchParams);
    fetch.flush();
  }, [fetch, search, searchParams]);

  const onFilterExpandedChange = (expanded: boolean) => {
    const nextParams = new URLSearchParams(searchParams);
    if (expanded) {
      nextParams.set('filterExpanded', 'true');
    } else {
      nextParams.delete('filterExpanded');
    }
    setSearchParams(nextParams, { replace: true });
  };

  return (
    <Box maxWidth={1200} margin="auto" padding={2}>

      {(!!search || labelFilters.length > 0 || data.terraformModules?.edges?.length !== 0) && <React.Fragment>
        <Typography variant="h5" sx={{ marginBottom: 2 }}>Terraform Modules</Typography>
        <Box marginBottom={2} display="flex" gap={1} alignItems="flex-start">
          <Box flex={1}>
            <SearchInput
              fullWidth
              defaultValue={search}
              placeholder="search for modules"
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
              {data.terraformModules.totalCount} module{data.terraformModules.totalCount === 1 ? '' : 's'}
            </Typography>
          </Box>
        </Paper>
        {(!data.terraformModules.edges || data.terraformModules.edges?.length === 0) && (!!search || labelFilters.length > 0) && <Typography
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
          No modules matching {search && labelFilters.length > 0 ? 'search and filters' : search ? `search "${search}"` : 'filters'}
        </Typography>}
        <InfiniteScroll
          dataLength={data.terraformModules.edges?.length ?? 0}
          next={() => loadNext(INITIAL_ITEM_COUNT)}
          hasMore={hasNext}
          loader={<ListSkeleton rowCount={3} />}
        >
          <List disablePadding sx={{ opacity: isRefreshing ? 0.5 : 1, transition: 'opacity 0.3s ease-in-out' }}>
            {data.terraformModules.edges?.map((edge) => edge?.node && <TerraformModuleSearchListItem
              key={edge.node.id}
              fragmentRef={edge.node}
            />)}
          </List>
        </InfiniteScroll>
      </React.Fragment>}

      {!search && labelFilters.length === 0 && data.terraformModules.edges?.length === 0 && <Box sx={{ marginTop: 4 }} display="flex" justifyContent="center">
        <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center" sx={{ maxWidth: 600 }}>
          <Typography variant="h6">You don't have access to any Terraform Modules</Typography>
        </Box>
      </Box>}
    </Box>
  );
}

export default TerraformModuleSearch;
