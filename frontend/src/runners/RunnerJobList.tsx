import { Box, Paper, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useMemo } from 'react';
import InfiniteScroll from 'react-infinite-scroll-component';
import { useLazyLoadQuery, usePaginationFragment, useSubscription } from 'react-relay/hooks';
import { useParams } from 'react-router-dom';
import { GraphQLSubscriptionConfig } from "relay-runtime";
import ListSkeleton from '../skeletons/ListSkeleton';
import RunnerJobListItem from './RunnerJobListItem';
import { RunnerJobListEventsSubscription } from './__generated__/RunnerJobListEventsSubscription.graphql';
import { RunnerJobListFragment_jobs$key } from './__generated__/RunnerJobListFragment_jobs.graphql';
import { RunnerJobListPaginationQuery } from './__generated__/RunnerJobListPaginationQuery.graphql';
import { RunnerJobListQuery } from './__generated__/RunnerJobListQuery.graphql';

const jobEventsSubscription = graphql`subscription RunnerJobListEventsSubscription($input: JobSubscriptionInput!) {
    jobEvents(input: $input) {
      action
      job {
        id
        ...RunnerJobListItemFragment
      }
    }
  }`;

const query = graphql`
    query RunnerJobListQuery($id: String!, $first: Int!, $after: String) {
        node(id: $id) {
            ... on Runner {
                id
                ...RunnerJobListFragment_jobs
            }
        }
    }`

function RunnerJobList() {
    const theme = useTheme();
    const runnerId = useParams().runnerId as string;

    const queryData = useLazyLoadQuery<RunnerJobListQuery>(query, { id: runnerId, first: 20 }, { fetchPolicy: 'store-and-network' });

    const { data, loadNext, hasNext } = usePaginationFragment<RunnerJobListPaginationQuery, RunnerJobListFragment_jobs$key>(
        graphql`
        fragment RunnerJobListFragment_jobs on Runner
        @refetchable(queryName: "RunnerJobListPaginationQuery") {
                type
                jobs(
                    first: $first
                    after: $after
                    sort: CREATED_AT_DESC
                ) @connection(key: "RunnerJobList_jobs") {
                    totalCount
                    edges {
                        node {
                            id
                            ...RunnerJobListItemFragment
                        }
                    }
                }
            }
        `, queryData.node
    );

    const jobSubscriptionConfig = useMemo<GraphQLSubscriptionConfig<RunnerJobListEventsSubscription>>(() => ({
        variables: { input: { runnerId } },
        subscription: jobEventsSubscription,
        onCompleted: () => console.log("Subscription completed"),
        onError: () => console.warn("Subscription error")
    }), [runnerId]);

    useSubscription<RunnerJobListEventsSubscription>(jobSubscriptionConfig);

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
                <Typography color="textSecondary">The following jobs have been claimed by this runner</Typography>
            </Box>
            {(!data?.jobs.edges || data?.jobs.edges?.length === 0) ? <Paper sx={{ p: 2, m: 2 }}>
                <Typography>This runner does not have any jobs.</Typography>
            </Paper>
                :
                <Box sx={{ p: 2 }}>
                    <Paper sx={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0, border: `1px solid ${theme.palette.divider}` }}>
                        <Box padding={2}>
                            <Typography variant="subtitle1">
                                {data?.jobs.totalCount} job{data?.jobs.totalCount !== 1 && 's'}
                            </Typography>
                        </Box>
                    </Paper>
                    <InfiniteScroll
                        dataLength={(data?.jobs.edges && data?.jobs.edges.length) ?? 0}
                        next={() => loadNext(20)}
                        hasMore={hasNext}
                        loader={<ListSkeleton rowCount={3} />}
                    >
                        <TableContainer sx={{
                            borderLeft: `1px solid ${theme.palette.divider}`,
                            borderRight: `1px solid ${theme.palette.divider}`,
                            borderBottom: `1px solid ${theme.palette.divider}`,
                            borderBottomLeftRadius: 4,
                            borderBottomRightRadius: 4,
                        }}>
                            <Table
                                sx={{ minWidth: 650, tableLayout: 'fixed' }}
                                aria-label="runner jobs"
                            >
                                <TableHead>
                                    <TableRow>
                                        <TableCell>Status</TableCell>
                                        <TableCell>ID</TableCell>
                                        <TableCell>Stage</TableCell>
                                        <TableCell>Workspace</TableCell>
                                        <TableCell>Duration</TableCell>
                                        <TableCell>Created</TableCell>
                                    </TableRow>
                                </TableHead>
                                <TableBody>
                                    {data.jobs.edges?.map((edge: any) => (
                                        <RunnerJobListItem
                                            key={edge.node.id}
                                            fragmentRef={edge.node}
                                        />
                                    ))}
                                </TableBody>
                            </Table>
                        </TableContainer>
                    </InfiniteScroll>
                </Box>}
        </Box>
    );
}

export default RunnerJobList
