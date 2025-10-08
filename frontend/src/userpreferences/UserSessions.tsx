import React, { useMemo } from 'react';
import { Box, Typography, Alert } from '@mui/material';
import { usePaginationFragment } from 'react-relay';
import graphql from 'babel-plugin-relay/macro';
import InfiniteScroll from 'react-infinite-scroll-component';
import UserSession from './UserSession';
import ListSkeleton from '../skeletons/ListSkeleton';
import { UserSessionsFragment_user$key } from './__generated__/UserSessionsFragment_user.graphql';
import { UserSessionsPaginationQuery } from './__generated__/UserSessionsPaginationQuery.graphql';

interface Props {
    fragmentRef: UserSessionsFragment_user$key;
}

function UserSessions({ fragmentRef }: Props) {
    const { data, loadNext, hasNext } = usePaginationFragment<UserSessionsPaginationQuery, UserSessionsFragment_user$key>(
        graphql`
            fragment UserSessionsFragment_user on User
            @refetchable(queryName: "UserSessionsPaginationQuery") {
                userSessions(
                    first: $first
                    after: $after
                    sort: CREATED_AT_DESC
                ) @connection(key: "UserSessions_userSessions") {
                    edges {
                        node {
                            id
                            ...UserSessionFragment_session
                        }
                    }
                }
            }
        `,
        fragmentRef
    );

    const sessions = useMemo(() =>
        data.userSessions.edges?.map(edge => edge?.node).filter((node): node is NonNullable<typeof node> => Boolean(node)) || [],
        [data.userSessions.edges]
    );

    return (
        <Box>
            <Typography variant="h6" gutterBottom>
                Sessions
            </Typography>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
                Manage your active login sessions. You can revoke any session to immediately log out from that device or browser.
            </Typography>

            {sessions.length === 0 ? (
                <Alert severity="info">
                    No active sessions found.
                </Alert>
            ) : (
                <InfiniteScroll
                    dataLength={sessions.length}
                    next={() => loadNext(10)}
                    hasMore={hasNext}
                    loader={<ListSkeleton rowCount={3} />}
                >
                    <Box>
                        {sessions.map(session => (
                            <UserSession
                                key={session.id}
                                fragmentRef={session}
                            />
                        ))}
                    </Box>
                </InfiniteScroll>
            )}
        </Box>
    );
}

export default UserSessions;
