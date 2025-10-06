import { Box, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useLazyLoadQuery, usePaginationFragment } from 'react-relay';
import ActivityEventList from '../activity/ActivityEventList';
import { HomeActivityFeedQuery } from './__generated__/HomeActivityFeedQuery.graphql';
import { HomeActivityFeedPaginationQuery } from './__generated__/HomeActivityFeedPaginationQuery.graphql';
import { HomeActivityFeedFragment_activity$key } from './__generated__/HomeActivityFeedFragment_activity.graphql';

const INITIAL_ITEM_COUNT = 20;

interface Props {
    username: string | undefined;
}

function HomeActivityFeed({ username }: Props) {
    const theme = useTheme();
    const queryData = useLazyLoadQuery<HomeActivityFeedQuery>(graphql`
        query HomeActivityFeedQuery($first: Int, $after: String, $username: String) {
           ...HomeActivityFeedFragment_activity
        }
    `, { first: INITIAL_ITEM_COUNT, username }, { fetchPolicy: 'store-and-network' });

    const { data, loadNext, hasNext } = usePaginationFragment<HomeActivityFeedPaginationQuery, HomeActivityFeedFragment_activity$key>(
        graphql`
      fragment HomeActivityFeedFragment_activity on Query
      @refetchable(queryName: "HomeActivityFeedPaginationQuery") {
        activityEvents(
            after: $after
            first: $first
            sort: CREATED_DESC
            username: $username
        ) @connection(key: "HomeActivityFeed_activityEvents") {
            edges {
                node {
                    id
                }
            }
            ...ActivityEventListFragment_connection
        }
      }
    `, queryData);

    const eventCount = data.activityEvents?.edges?.length || 0;

    return (
        <Box>
            {eventCount > 0 &&
                <ActivityEventList
                    loadNext={loadNext}
                    hasNext={hasNext}
                    fragmentRef={data.activityEvents}
                />
            }
            {eventCount === 0 && <Box
                border={`1px dashed ${theme.palette.divider}`}
                borderRadius={2}
                mt={4}
                display="flex"
                justifyContent="center"
            >
                <Box
                    padding={4}
                    display="flex"
                    flexDirection="column"
                    justifyContent="center"
                    alignItems="center"
                    sx={{ maxWidth: 600 }}
                >
                    <Typography variant="subtitle1" color="textSecondary">You do not have any activity events</Typography>
                </Box>
            </Box>}
        </Box>
    );
}

export default HomeActivityFeed;
