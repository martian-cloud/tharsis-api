import { Box, Paper, Table, TableBody, TableContainer, TableCell, TableHead, TableRow, Typography, useTheme } from "@mui/material";
import graphql from 'babel-plugin-relay/macro';
import TeamMemberListItem from './TeamMemberListItem';
import InfiniteScroll from 'react-infinite-scroll-component';
import ListSkeleton from '../skeletons/ListSkeleton';
import { usePaginationFragment } from 'react-relay/hooks';
import { TeamMemberListPaginationQuery } from "./__generated__/TeamMemberListPaginationQuery.graphql";
import { TeamMemberListFragment_members$key } from './__generated__/TeamMemberListFragment_members.graphql';

interface Props {
    fragmentRef: TeamMemberListFragment_members$key
}

function TeamMemberList({ fragmentRef }: Props) {
    const theme = useTheme();

    const { data, loadNext, hasNext } = usePaginationFragment<TeamMemberListPaginationQuery, TeamMemberListFragment_members$key>(graphql`
        fragment TeamMemberListFragment_members on Team
        @refetchable(queryName: "TeamMemberListPaginationQuery") {
            id
            members(
                first: $first
                after: $after
                ) @connection(key: "TeamMemberList_members"){
                edges {
                    node {
                        id
                        ...TeamMemberListItemFragment_member
                    }
                }
            }
        }
    `, fragmentRef);

    if (data.members?.edges && data.members?.edges.length > 0) {
        return (
            <InfiniteScroll
                dataLength={data.members?.edges.length}
                next={() => loadNext(20)}
                hasMore={hasNext}
                loader={<ListSkeleton rowCount={3} />}
            >
                <TableContainer>
                    <Table
                        sx={{
                            minWidth: 350,
                            borderCollapse: 'separate',
                            borderSpacing: 0,
                            'td, th': {
                                borderBottom: `1px solid ${theme.palette.divider}`,
                            },
                            'tr:first-of-type th': {
                                borderTop: `1px solid ${theme.palette.divider}`,
                            },
                            'th:first-of-type': {
                                borderTopLeftRadius: 4,
                            },
                            'th:last-of-type': {
                                borderTopRightRadius: 4,
                            },
                            'td:first-of-type, th:first-of-type': {
                                borderLeft: `1px solid ${theme.palette.divider}`,
                            },
                            'td:last-of-type, th:last-of-type': {
                                borderRight: `1px solid ${theme.palette.divider}`,
                            },
                            'tr:last-of-type td:first-of-type': {
                                borderBottomLeftRadius: 4,
                            },
                            'tr:last-of-type td:last-of-type': {
                                borderBottomRightRadius: 4
                            }
                        }}
                        aria-label="TeamMembers">
                        <TableHead>
                            <TableRow>
                                <TableCell>Name</TableCell>
                                <TableCell>Team Maintainer</TableCell>
                                <TableCell>Last Updated</TableCell>
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {data.members?.edges?.map((edge: any) => <TeamMemberListItem
                                key={edge.node.id}
                                fragmentRef={edge.node}
                            />)}
                        </TableBody>
                    </Table>
                </TableContainer>
            </InfiniteScroll>
        );
    } else {
        return <Paper variant="outlined" sx={{ display: "flex", justifyContent: "center" }}>
            <Box sx={{ p: 4 }}>
                <Typography variant="h6" color="textSecondary" align="center">There are no members on this team.</Typography>
            </Box>
        </Paper>;
    }
}

export default TeamMemberList;
