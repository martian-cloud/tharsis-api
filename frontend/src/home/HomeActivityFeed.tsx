import { Box, ToggleButton, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import React, { useContext } from 'react';
import { useLazyLoadQuery, usePaginationFragment } from 'react-relay';
import { useSearchParams } from 'react-router';
import ActivityEventList from '../activity/ActivityEventList';
import { UserContext } from '../UserContext';
import { HomeActivityFeedFragment_activity$key } from './__generated__/HomeActivityFeedFragment_activity.graphql';
import { HomeActivityFeedPaginationQuery } from './__generated__/HomeActivityFeedPaginationQuery.graphql';
import { HomeActivityFeedQuery } from './__generated__/HomeActivityFeedQuery.graphql';

const INITIAL_ITEM_COUNT = 20;

function HomeActivityFeed() {
    const user = useContext(UserContext);
    const [searchParams, setSearchParams] = useSearchParams();

    // This filter will only show the current users activity when true
    const showMyActivity = searchParams.get('filter') === 'myActivity';

    const queryData = useLazyLoadQuery<HomeActivityFeedQuery>(graphql`
        query HomeActivityFeedQuery($first: Int, $after: String, $username: String) {
           ...HomeActivityFeedFragment_activity
        }
    `, { first: INITIAL_ITEM_COUNT, username: showMyActivity ? user.username : undefined }, { fetchPolicy: 'store-and-network' });

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
        <React.Fragment>
            {eventCount === 0 && <Box flex={1} mt={4} display="flex" justifyContent="center">
                <Box
                    sx={{
                        p: 4,
                        maxWidth: 600,
                        display: 'flex',
                        flexDirection: 'column',
                        alignItems: 'center',
                        justifyContent: 'center',
                    }}>
                    <Typography variant="h6">Welcome to Tharsis!</Typography>
                    <Typography color="textSecondary" align="center" sx={{ marginBottom: 2 }}>
                        Get started using Tharsis to manage your Terraform deployments
                    </Typography>
                </Box>
            </Box>}
            {eventCount > 0 && <React.Fragment>
                <Box display="flex" justifyContent="space-between" alignItems="center">
                    <Typography variant="h6" fontWeight={600} mb={1}>Activity</Typography>
                    <ToggleButton
                        onChange={() => setSearchParams(!showMyActivity ? { filter: 'myActivity' } : {}, { replace: true })}
                        color="secondary"
                        selected={showMyActivity}
                        size="small"
                        value="myActivity">
                        My Activity
                    </ToggleButton>
                </Box>
                <ActivityEventList
                    loadNext={loadNext}
                    hasNext={hasNext}
                    fragmentRef={data.activityEvents}
                />
            </React.Fragment>}
        </React.Fragment>
    );
}

export default HomeActivityFeed;
