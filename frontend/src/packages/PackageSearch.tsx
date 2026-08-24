import Box from '@mui/material/Box';
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
import ListSkeleton from '../skeletons/ListSkeleton';
import { PageLayoutProvider } from '../layout/PageLayoutContext';
import PackageSearchListItem from './PackageSearchListItem';
import { PackageSearchFragment_packages$key } from './__generated__/PackageSearchFragment_packages.graphql';
import { PackageSearchPaginationQuery } from './__generated__/PackageSearchPaginationQuery.graphql';
import { PackageSearchQuery } from './__generated__/PackageSearchQuery.graphql';

export const INITIAL_ITEM_COUNT = 50;

const query = graphql`
    query PackageSearchQuery($first: Int, $last: Int, $after: String, $before: String, $search: String) {
      ...PackageSearchFragment_packages
    }
`;

interface Props {
  queryRef: PreloadedQuery<PackageSearchQuery>
  search?: string
}

function PackageSearch({ search = '', queryRef }: Props) {
  const queryData = usePreloadedQuery<PackageSearchQuery>(query, queryRef);
  const [searchParams, setSearchParams] = useSearchParams();
  const theme = useTheme();
  const environment = useRelayEnvironment();

  const [isRefreshing, setIsRefreshing] = useState(false);

  const { data, loadNext, hasNext, refetch } = usePaginationFragment<PackageSearchPaginationQuery, PackageSearchFragment_packages$key>(
    graphql`
    fragment PackageSearchFragment_packages on Query
    @refetchable(queryName: "PackageSearchPaginationQuery") {
      packages(
          after: $after
          before: $before
          first: $first
          last: $last
          search: $search
          sort: NAME_ASC
      ) @connection(key: "PackageSearch_packages") {
          totalCount
          edges {
              node {
                  id
                  ...PackageSearchListItemFragment_package
              }
          }
      }
    }
  `, queryData);

  const fetch = useMemo(
    () =>
      throttle(
        (input: string, existingSearchParams: URLSearchParams) => {
          setIsRefreshing(true);

          fetchQuery(environment, query, {
            first: INITIAL_ITEM_COUNT,
            search: input,
          })
            .subscribe({
              complete: () => {
                const nextParams = new URLSearchParams(existingSearchParams);
                if (input.trim() !== '') {
                  nextParams.set('search', input);
                } else {
                  nextParams.delete('search');
                }

                setSearchParams(nextParams, { replace: true });

                // *After* the query has been fetched, we call refetch again to re-render with the
                // updated data. At this point the data for the query should be cached, so we use
                // the 'store-only' fetchPolicy to avoid suspending.
                refetch({
                  first: INITIAL_ITEM_COUNT,
                  search: input,
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
    fetch(newSearch, searchParams);
  }, [fetch, searchParams]);

  const edges = data.packages?.edges ?? [];

  return (
    <PageLayoutProvider>

      {(!!search || edges.length !== 0) && <React.Fragment>
        <Typography variant="h5" sx={{ marginBottom: 2 }}>Packages</Typography>
        <Box marginBottom={2} display="flex" gap={1} alignItems="flex-start">
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
              {data.packages.totalCount} package{data.packages.totalCount === 1 ? '' : 's'}
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
          No packages matching search &quot;{search}&quot;
        </Typography>}
        <InfiniteScroll
          dataLength={edges.length}
          next={() => loadNext(INITIAL_ITEM_COUNT)}
          hasMore={hasNext}
          loader={<ListSkeleton rowCount={3} />}
        >
          <List disablePadding sx={{ opacity: isRefreshing ? 0.5 : 1, transition: 'opacity 0.3s ease-in-out' }}>
            {edges.map((edge) => edge?.node && <PackageSearchListItem
              key={edge.node.id}
              fragmentRef={edge.node}
            />)}
          </List>
        </InfiniteScroll>
      </React.Fragment>}

      {!search && edges.length === 0 && <Box sx={{ marginTop: 4 }} display="flex" justifyContent="center">
        <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center" sx={{ maxWidth: 600 }}>
          <Typography variant="h6">You don't have access to any Packages</Typography>
        </Box>
      </Box>}
    </PageLayoutProvider>
  );
}

export default PackageSearch;
