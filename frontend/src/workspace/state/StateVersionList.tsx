import { Box, Paper, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import InfiniteScroll from 'react-infinite-scroll-component';
import { useFragment, useLazyLoadQuery, usePaginationFragment } from 'react-relay';
import ListSkeleton from '../../skeletons/ListSkeleton';
import StateVersionListItem from './StateVersionListItem';
import { StateVersionListFragment_workspace$key } from './__generated__/StateVersionListFragment_workspace.graphql';
import { StateVersionListFragment_stateVersions$key } from './__generated__/StateVersionListFragment_stateVersions.graphql';
import { StateVersionListPaginationQuery } from './__generated__/StateVersionListPaginationQuery.graphql';
import { StateVersionListQuery } from './__generated__/StateVersionListQuery.graphql';

const query = graphql`
    query StateVersionListQuery($first: Int, $last: Int, $after: String, $before: String, $workspaceId: String!) {
        node(id: $workspaceId) {
            ...on Workspace {
                ...StateVersionListFragment_stateVersions
            }
        }
    }
`;

interface Props {
    fragmentRef: StateVersionListFragment_workspace$key
}

function StateVersionList({ fragmentRef }: Props) {

    const workspace = useFragment<StateVersionListFragment_workspace$key>(
        graphql`
        fragment StateVersionListFragment_workspace on Workspace {
            id
            fullPath
        }
        `, fragmentRef
    );

    const queryData = useLazyLoadQuery<StateVersionListQuery>(query, { first: 100, workspaceId: workspace.id }, { fetchPolicy: 'store-and-network' });

    const { data, loadNext, hasNext } = usePaginationFragment<StateVersionListPaginationQuery, StateVersionListFragment_stateVersions$key>(
        graphql`
        fragment StateVersionListFragment_stateVersions on Workspace
        @refetchable(queryName: "StateVersionListPaginationQuery") {
            stateVersions(
                after: $after
                before: $before
                first: $first
                last: $last
                sort: UPDATED_AT_ASC
            ) @connection(key: "StateVersionList_stateVersions") {
                edges {
                    node {
                        id
                        ...StateVersionListItemFragment_stateVersion
                    }
                }
            }
        }
        `, queryData.node);

    return (
        <Box>
            {data?.stateVersions.edges && data?.stateVersions.edges.length > 0 && <InfiniteScroll
                dataLength={data?.stateVersions.edges?.length ?? 0}
                next={() => loadNext(20)}
                hasMore={hasNext}
                loader={<ListSkeleton rowCount={3} />}
            >
                <TableContainer>
                    <Table sx={{ minWidth: 650 }} aria-label="workspace state versions">
                        <TableHead>
                            <TableRow>
                                <TableCell>State Version ID</TableCell>
                                <TableCell>Run ID</TableCell>
                                <TableCell>Created At</TableCell>
                                <TableCell>Created By</TableCell>
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {data.stateVersions.edges.map((edge: any) => (
                                <StateVersionListItem key={edge.node.id} stateVersionKey={edge.node} workspacePath={workspace.fullPath} />
                            ))}
                        </TableBody>
                    </Table>
                </TableContainer>
            </InfiniteScroll>}
            {data?.stateVersions.edges?.length === 0 && <Paper variant="outlined" sx={{ marginTop: 4, display: 'flex', justifyContent: 'center' }}>
                <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center">
                    <Typography variant="h6" color="textSecondary" align="center">No state versions have been created in this workspace</Typography>
                </Box>
            </Paper>}
        </Box>
    );
}

export default StateVersionList;
