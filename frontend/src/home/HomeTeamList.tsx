import { Box, Link, List, ListItem, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useLazyLoadQuery, usePaginationFragment } from 'react-relay/hooks';
import HomeTeamListItem from './HomeTeamListItem';
import { HomeTeamListFragment_teams$key } from './__generated__/HomeTeamListFragment_teams.graphql';
import { HomeTeamListPaginationQuery } from './__generated__/HomeTeamListPaginationQuery.graphql';
import { HomeTeamListQuery } from './__generated__/HomeTeamListQuery.graphql';

const INITIAL_ITEM_COUNT = 5;

const query = graphql`
    query HomeTeamListQuery($first: Int!, $after: String) {
        me {
            ...on User {
                ...HomeTeamListFragment_teams
            }
        }
    }
`;

function HomeTeamList() {
    const theme = useTheme();
    const queryData = useLazyLoadQuery<HomeTeamListQuery>(query, { first: INITIAL_ITEM_COUNT }, { fetchPolicy: 'store-and-network' });

    const { data, loadNext, hasNext } = usePaginationFragment<HomeTeamListPaginationQuery, HomeTeamListFragment_teams$key>(
        graphql`
        fragment HomeTeamListFragment_teams on User
        @refetchable(queryName: "HomeTeamListPaginationQuery") {
            teams(
                first: $first
                after: $after
                sort: NAME_ASC
                ) @connection(key: "HomeTeamList_teams") {
                    totalCount
                    edges {
                        node {
                            id
                            ...HomeTeamListItemFragment_team
                        }
                    }
                }
            }
        `, queryData.me);

    const edgeCount = data?.teams?.edges?.length ?? 0;

    return (
        <Box>
            {edgeCount === 0 && <Box sx={{ p: 2, border: `1px dashed ${theme.palette.divider}`, borderRadius: 2 }}>
                <Typography color="textSecondary" variant="body2">You are not currently a member of any teams.</Typography>
            </Box>}
            {edgeCount > 0 && <List disablePadding>
                {data?.teams?.edges?.map((edge: any, index: number) => <HomeTeamListItem
                    key={edge.node.id}
                    fragmentRef={edge.node}
                    last={index === (edgeCount - 1)}
                />)}
                {hasNext && <ListItem>
                    <Link
                        variant="body2"
                        color="textSecondary"
                        sx={{ cursor: 'pointer' }}
                        underline="hover"
                        onClick={() => loadNext(INITIAL_ITEM_COUNT)}
                    >
                        Show more
                    </Link>
                </ListItem>}
            </List>}
        </Box>
    );
}

export default HomeTeamList;
