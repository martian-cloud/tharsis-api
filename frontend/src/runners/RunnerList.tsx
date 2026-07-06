import { Box, Button, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import InfiniteScroll from 'react-infinite-scroll-component';
import { LoadMoreFn, useFragment } from 'react-relay/hooks';
import { Link as RouterLink } from 'react-router-dom';
import { ResponsiveTable } from '../common/ResponsiveTable';
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
                    <ResponsiveTable
                        ariaLabel="runners"
                        minWidth={650}
                        columns={[{ label: 'Name' }, { label: 'Status' }, { label: 'Created' }, { label: 'Last Updated' }]}
                    >
                        {data.edges?.map((edge: any) => (
                            <RunnerListItem key={edge.node.id} fragmentRef={edge.node} inherited={!!groupPath && groupPath !== edge.node.groupPath} />
                        ))}
                    </ResponsiveTable>
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
