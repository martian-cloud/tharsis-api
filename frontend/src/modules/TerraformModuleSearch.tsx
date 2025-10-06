import Box from '@mui/material/Box';
import List from '@mui/material/List';
import Paper from '@mui/material/Paper';
import { useTheme } from '@mui/material/styles';
import Typography from '@mui/material/Typography';
import graphql from 'babel-plugin-relay/macro';
import throttle from 'lodash.throttle';
import React, { useMemo, useState } from 'react';
import InfiniteScroll from 'react-infinite-scroll-component';
import { fetchQuery, PreloadedQuery, usePaginationFragment, usePreloadedQuery, useRelayEnvironment } from 'react-relay/hooks';
import SearchInput from '../common/SearchInput';
import ListSkeleton from '../skeletons/ListSkeleton';
import TerraformModuleSearchListItem from './TerraformModuleSearchListItem';
import { TerraformModuleSearchFragment_modules$key } from './__generated__/TerraformModuleSearchFragment_modules.graphql';
import { TerraformModuleSearchPaginationQuery } from './__generated__/TerraformModuleSearchPaginationQuery.graphql';
import { TerraformModuleSearchQuery } from './__generated__/TerraformModuleSearchQuery.graphql';

export const INITIAL_ITEM_COUNT = 50;

const query = graphql`
    query TerraformModuleSearchQuery($first: Int, $last: Int, $after: String, $before: String, $search: String) {
      ...TerraformModuleSearchFragment_modules
    }
`;

interface Props {
  queryRef: PreloadedQuery<TerraformModuleSearchQuery>
}

function TerraformModuleSearch(props: Props) {
  const queryData = usePreloadedQuery<TerraformModuleSearchQuery>(query, props.queryRef);

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

  const [search, setSearch] = useState<string | undefined>('');
  const [isRefreshing, setIsRefreshing] = useState(false);

  const environment = useRelayEnvironment();
  const theme = useTheme();

  const fetch = useMemo(
    () =>
      throttle(
        (input?: string) => {
          setIsRefreshing(true);

          fetchQuery(environment, query, { first: INITIAL_ITEM_COUNT, search: input })
            .subscribe({
              complete: () => {
                setIsRefreshing(false);
                setSearch(input);

                // *After* the query has been fetched, we call
                // refetch again to re-render with the updated data.
                // At this point the data for the query should
                // be cached, so we use the 'store-only'
                // fetchPolicy to avoid suspending.
                refetch({ first: INITIAL_ITEM_COUNT, search: input }, { fetchPolicy: 'store-only' });
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

  const onSearchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    fetch(event.target.value.toLowerCase());
  };

  const onKeyPress = (event: React.KeyboardEvent<HTMLInputElement>) => {
    // Only handle enter key type
    if (event.which === 13) {
      fetch.flush();
    }
  };

  return (
    <Box maxWidth={1200} margin="auto" padding={2}>

      {(search !== '' || data.terraformModules?.edges?.length !== 0) && <React.Fragment>
        <Typography variant="h5" sx={{ marginBottom: 2 }}>Terraform Modules</Typography>
        <Box marginBottom={2}>
          <SearchInput
            fullWidth
            placeholder="search for modules"
            onChange={onSearchChange}
            onKeyPress={onKeyPress}
          />
        </Box>
        <Paper sx={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0, border: `1px solid ${theme.palette.divider}` }}>
          <Box padding={2} display="flex" alignItems="center" justifyContent="space-between">
            <Typography variant="subtitle1">
              {data.terraformModules.totalCount} module{data.terraformModules.totalCount === 1 ? '' : 's'}
            </Typography>
          </Box>
        </Paper>
        {(!data.terraformModules.edges || data.terraformModules.edges?.length === 0) && search !== '' && <Typography
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
          No modules matching search <strong>{search}</strong>
        </Typography>}
        <InfiniteScroll
          dataLength={data.terraformModules.edges?.length ?? 0}
          next={() => loadNext(INITIAL_ITEM_COUNT)}
          hasMore={hasNext}
          loader={<ListSkeleton rowCount={3} />}
        >
          <List disablePadding sx={isRefreshing ? { opacity: 0.5 } : null}>
            {data.terraformModules.edges?.map((edge: any) => <TerraformModuleSearchListItem
              key={edge.node.id}
              fragmentRef={edge.node}
            />)}
          </List>
        </InfiniteScroll>
      </React.Fragment>}

      {!search && data.terraformModules.edges?.length === 0 && <Box sx={{ marginTop: 4 }} display="flex" justifyContent="center">
        <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center" sx={{ maxWidth: 600 }}>
          <Typography variant="h6">You don't have access to any Terraform Modules</Typography>
        </Box>
      </Box>}
    </Box>
  );
}

export default TerraformModuleSearch;
