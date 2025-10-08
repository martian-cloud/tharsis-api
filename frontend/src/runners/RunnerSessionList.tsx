import { Box, List, Paper, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useState } from 'react';
import InfiniteScroll from 'react-infinite-scroll-component';
import { ConnectionHandler, useLazyLoadQuery, usePaginationFragment } from 'react-relay/hooks';
import { useParams } from 'react-router-dom';
import ListSkeleton from '../skeletons/ListSkeleton';
import RunnerSessionErrorLogDialog from './RunnerSessionErrorLogDialog';
import RunnerSessionListItem from './RunnerSessionListItem';
import { RunnerSessionListFragment_sessions$key } from './__generated__/RunnerSessionListFragment_sessions.graphql';
import { RunnerSessionListPaginationQuery } from './__generated__/RunnerSessionListPaginationQuery.graphql';
import { RunnerSessionListQuery } from './__generated__/RunnerSessionListQuery.graphql';

export function GetConnections(runnerId: string): [string] {
    const connectionId = ConnectionHandler.getConnectionID(
        runnerId,
        "RunnerSessionList_sessions",
        { sort: "LAST_CONTACTED_AT_DESC" }
    );
    return [connectionId];
}

const query = graphql`
    query RunnerSessionListQuery($id: String!, $first: Int!, $after: String) {
        node(id: $id) {
            ... on Runner {
                id
                ...RunnerSessionListFragment_sessions
            }
        }
    }`

function RunnerSessionList() {
    const theme = useTheme();
    const runnerId = useParams().runnerId as string;

    const [selectedSession, setSelectedSession] = useState(null);

    const queryData = useLazyLoadQuery<RunnerSessionListQuery>(query, { id: runnerId, first: 20 }, { fetchPolicy: 'store-and-network' });

    const { data, loadNext, hasNext } = usePaginationFragment<RunnerSessionListPaginationQuery, RunnerSessionListFragment_sessions$key>(
        graphql`
        fragment RunnerSessionListFragment_sessions on Runner
        @refetchable(queryName: "RunnerSessionListPaginationQuery") {
                type
                sessions(
                    first: $first
                    after: $after
                    sort: LAST_CONTACTED_AT_DESC
                ) @connection(key: "RunnerSessionList_sessions") {
                    totalCount
                    edges {
                        node {
                            id
                            ...RunnerSessionListItemFragment
                        }
                    }
                }
            }
        `, queryData.node
    );

    return (
        <Box sx={{ border: 1, borderTop: 0, borderBottomLeftRadius: 4, borderBottomRightRadius: 4, borderColor: 'divider' }}>
            <Box sx={{
                display: 'flex',
                flexDirection: 'row',
                justifyContent: 'space-between',
                alignItems: 'flex-start',
                p: 2,
                pb: 0,
                [theme.breakpoints.down('lg')]: {
                    flexDirection: 'column',
                    alignItems: 'flex-start',
                    '& > *': { mb: 2 }
                }
            }}>
                <Typography color="textSecondary">Runner sessions are individual connections from this runner</Typography>
            </Box>
            {(!data?.sessions.edges || data?.sessions.edges?.length === 0) ? (
                <Paper sx={{ p: 2, m: 2 }}>
                    <Typography>This runner does not have any sessions.</Typography>
                </Paper>
            )
                :
                (
                    <Box sx={{ p: 2 }}>
                        <Paper sx={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0, border: `1px solid ${theme.palette.divider}` }}>
                            <Box padding={2}>
                                <Typography variant="subtitle1">
                                    {data?.sessions.totalCount} session{data?.sessions.totalCount !== 1 && 's'}
                                </Typography>
                            </Box>
                        </Paper>
                        <InfiniteScroll
                            dataLength={(data?.sessions.edges && data?.sessions.edges.length) ?? 0}
                            next={() => loadNext(20)}
                            hasMore={hasNext}
                            loader={<ListSkeleton rowCount={3} />}
                        >
                            <List disablePadding>{data?.sessions.edges?.map((edge: any) => <RunnerSessionListItem
                                key={edge.node.id}
                                fragmentRef={edge.node}
                                onClick={() => setSelectedSession(edge.node.id)}
                            />)}
                            </List>
                        </InfiniteScroll>
                    </Box>
                )}
            {selectedSession && <RunnerSessionErrorLogDialog sessionId={selectedSession} onClose={() => setSelectedSession(null)} />}
        </Box>
    );
}

export default RunnerSessionList
