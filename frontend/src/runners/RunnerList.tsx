import { Box, Button, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import InfiniteScroll from 'react-infinite-scroll-component';
import { LoadMoreFn, useFragment } from 'react-relay/hooks';
import { Link as RouterLink } from 'react-router-dom';
import ListSkeleton from '../skeletons/ListSkeleton';
import RunnerListItem from './RunnerListItem';
import { RunnerListFragment_runners$key } from './__generated__/RunnerListFragment_runners.graphql';

const DESCRIPTION = 'Runners are responsible for claiming and launching jobs.';

interface Props {
    fragmentRef: RunnerListFragment_runners$key
    loadNext: LoadMoreFn<any>
    hasNext: boolean
    hideNewRunnerButton?: boolean
    groupPath?: string
}

function RunnerList({ fragmentRef, loadNext, hasNext, hideNewRunnerButton, groupPath }: Props) {
    const theme = useTheme();

    const data = useFragment<RunnerListFragment_runners$key>(graphql`
        fragment RunnerListFragment_runners on RunnerConnection {
            edges {
                node {
                    id
                    groupPath
                    ...RunnerListItemFragment_runner
                }
            }
        }
    `, fragmentRef);

    return (
        <Box>
            {data.edges?.length !== 0 ? <Box>
                <Box>
                    <Box sx={{
                        display: 'flex',
                        flexDirection: 'row',
                        justifyContent: 'space-between',
                        mb: 2,
                        [theme.breakpoints.down('md')]: {
                            flexDirection: 'column',
                            alignItems: 'flex-start',
                            '& > *': { mb: 2 },
                        }
                    }}>
                        <Box>
                            <Typography variant="h5" gutterBottom>Runners</Typography>
                            <Typography variant="body2">
                                {DESCRIPTION}
                            </Typography>
                        </Box>
                        {!hideNewRunnerButton && <Box>
                            <Button
                                sx={{ minWidth: 125 }}
                                component={RouterLink}
                                variant="outlined"
                                to="new"
                            >
                                New Runner
                            </Button>
                        </Box>}
                    </Box>
                </Box>
                <InfiniteScroll
                    dataLength={data.edges?.length ?? 0}
                    next={() => loadNext(20)}
                    hasMore={hasNext}
                    loader={<ListSkeleton rowCount={3} />}
                >
                    <TableContainer>
                        <Table
                            sx={{ minWidth: 650, tableLayout: 'fixed' }}
                            aria-label="runners"
                        >
                            <TableHead>
                                <TableRow>
                                    <TableCell>Name</TableCell>
                                    <TableCell>Status</TableCell>
                                    <TableCell>Created By</TableCell>
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {data.edges?.map((edge: any) => (
                                    <RunnerListItem key={edge.node.id} fragmentRef={edge.node} inherited={!!groupPath && groupPath !== edge.node.groupPath} />
                                ))}
                            </TableBody>
                        </Table>
                    </TableContainer>
                </InfiniteScroll>
            </Box> : <Box sx={{ mt: 4 }} display="flex" justifyContent="center">
                <Box p={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center" sx={{ maxWidth: 600 }}>
                    <Typography variant="h6">Get started with runners</Typography>
                    <Typography sx={{ mb: 2 }} color="textSecondary" align="center" >
                        {DESCRIPTION}
                    </Typography>
                    {!hideNewRunnerButton && <Button component={RouterLink} variant="outlined" to="new">New Runner</Button>}
                </Box>
            </Box>}
        </Box>
    );
}

export default RunnerList;
