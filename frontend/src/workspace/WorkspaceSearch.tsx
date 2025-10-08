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
import WorkspaceSearchListItem from './WorkspaceSearchListItem';
import { WorkspaceSearchFragment_workspaces$key } from './__generated__/WorkspaceSearchFragment_workspaces.graphql';
import { WorkspaceSearchPaginationQuery } from './__generated__/WorkspaceSearchPaginationQuery.graphql';
import { WorkspaceSearchQuery } from './__generated__/WorkspaceSearchQuery.graphql';

export const INITIAL_ITEM_COUNT = 100;

const query = graphql`
    query WorkspaceSearchQuery($first: Int, $last: Int, $after: String, $before: String, $search: String) {
      ...WorkspaceSearchFragment_workspaces
    }
`;

interface Props {
  queryRef: PreloadedQuery<WorkspaceSearchQuery>
}

function WorkspaceSearch(props: Props) {
  const queryData = usePreloadedQuery<WorkspaceSearchQuery>(query, props.queryRef);

  const theme = useTheme();

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

  const [search, setSearch] = useState<string | undefined>('');
  const [isRefreshing, setIsRefreshing] = useState(false);

  const environment = useRelayEnvironment();

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

      {(search !== '' || data.workspaces?.edges?.length !== 0) && <React.Fragment>
        <Typography variant="h5" sx={{ marginBottom: 2 }}>Workspaces</Typography>
        <Box marginBottom={2}>
          <SearchInput
            fullWidth
            placeholder="search for workspaces"
            onChange={onSearchChange}
            onKeyPress={onKeyPress}
          />
        </Box>
        <Paper sx={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0, border: `1px solid ${theme.palette.divider}` }}>
          <Box padding={2} display="flex" alignItems="center" justifyContent="space-between">
            <Typography variant="subtitle1">
              {data.workspaces.totalCount} workspace{data.workspaces.totalCount === 1 ? '' : 's'}
            </Typography>
          </Box>
        </Paper>
        {(!data.workspaces.edges || data.workspaces.edges?.length === 0) && search !== '' && <Typography
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
          No workspaces matching search <strong>{search}</strong>
        </Typography>}
        <InfiniteScroll
          dataLength={data.workspaces.edges?.length ?? 0}
          next={() => loadNext(INITIAL_ITEM_COUNT)}
          hasMore={hasNext}
          loader={<ListSkeleton rowCount={3} />}
        >
          <List disablePadding sx={isRefreshing ? { opacity: 0.5 } : null}>
            {data.workspaces.edges?.map((edge: any) => <WorkspaceSearchListItem
              key={edge.node.id}
              workspaceKey={edge.node}
            />)}
          </List>
        </InfiniteScroll>
      </React.Fragment>}

      {!search && data.workspaces.edges?.length === 0 && <Box sx={{ marginTop: 4 }} display="flex" justifyContent="center">
        <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center" sx={{ maxWidth: 600 }}>
          <Typography variant="h6">You don't have access to any workspaces</Typography>
        </Box>
      </Box>}
    </Box>
  );
}

export default WorkspaceSearch;
