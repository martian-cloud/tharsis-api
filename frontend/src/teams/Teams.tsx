import { Box, List, Paper, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import throttle from 'lodash/throttle';
import React, { Suspense, useMemo, useState } from 'react';
import InfiniteScroll from 'react-infinite-scroll-component';
import { fetchQuery, useLazyLoadQuery, usePaginationFragment, useRelayEnvironment } from 'react-relay/hooks';
import SearchInput from '../common/SearchInput';
import ListSkeleton from '../skeletons/ListSkeleton';
import TeamListItem from './TeamListItem';
import { TeamsFragment_teams$key } from './__generated__/TeamsFragment_teams.graphql';
import { TeamsPaginationQuery } from './__generated__/TeamsPaginationQuery.graphql';
import { TeamsQuery } from './__generated__/TeamsQuery.graphql';

const INITIAL_ITEM_COUNT = 100;

const query = graphql`
    query TeamsQuery($first: Int, $after: String, $search: String) {
        ...TeamsFragment_teams
    }
`;

function TeamList() {
    const theme = useTheme();
    const environment = useRelayEnvironment();
    const [search, setSearch] = useState('');

    const queryData = useLazyLoadQuery<TeamsQuery>(query, { first: INITIAL_ITEM_COUNT }, { fetchPolicy: 'store-and-network' });

    const { data, loadNext, hasNext, refetch } = usePaginationFragment<TeamsPaginationQuery, TeamsFragment_teams$key>(
        graphql`
        fragment TeamsFragment_teams on Query
        @refetchable(queryName: "TeamsPaginationQuery") {
            teams(
                after: $after
                first: $first
                search: $search
                sort: NAME_ASC
            ) @connection(key: "TeamList_teams") {
                totalCount
                edges {
                    node {
                        id
                        ...TeamListItemFragment_team
                    }
                }
            }
        }
    `, queryData);

    const fetch = useMemo(
        () =>
            throttle(
                (input?: string) => {
                    fetchQuery(environment, query, { first: INITIAL_ITEM_COUNT, search: input })
                        .subscribe({
                            complete: () => {
                                setSearch(input ?? '');
                                refetch({ first: INITIAL_ITEM_COUNT, search: input }, { fetchPolicy: 'store-only' });
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

    const onKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
        if (event.key === 'Enter') {
            fetch.flush();
        }
    };

    const edges = data.teams?.edges ?? [];

    return (
        <Box maxWidth={1200} margin="auto" padding={2}>
            <Typography variant="h5" sx={{ marginBottom: 2 }}>Teams</Typography>
            <SearchInput
                sx={{ marginBottom: 2 }}
                fullWidth
                placeholder="search for teams"
                onChange={onSearchChange}
                onKeyDown={onKeyDown}
            />
            <Paper sx={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0, border: `1px solid ${theme.palette.divider}` }}>
                <Box padding={2} display="flex" alignItems="center" justifyContent="space-between">
                    <Typography variant="subtitle1">
                        {data.teams?.totalCount ?? 0} team{data.teams?.totalCount === 1 ? '' : 's'}
                    </Typography>
                </Box>
            </Paper>
            {edges.length === 0 && <Typography
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
                {search ? `No teams matching "${search}"` : 'No teams found'}
            </Typography>}
            <InfiniteScroll
                dataLength={edges.length}
                next={() => loadNext(INITIAL_ITEM_COUNT)}
                hasMore={hasNext}
                loader={<ListSkeleton rowCount={3} />}
            >
                <List disablePadding>
                    {edges.map((edge: any) => <TeamListItem
                        key={edge.node.id}
                        fragmentRef={edge.node}
                    />)}
                </List>
            </InfiniteScroll>
        </Box>
    );
}

function Teams() {
    return (
        <Suspense fallback={<Box maxWidth={1200} margin="auto" padding={2}>
            <ListSkeleton rowCount={5} />
        </Box>}>
            <TeamList />
        </Suspense>
    );
}

export default Teams;
