import { List, useMediaQuery, useTheme } from '@mui/material';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import graphql from 'babel-plugin-relay/macro';
import InfiniteScroll from 'react-infinite-scroll-component';
import { useFragment } from "react-relay/hooks";
import ListSkeleton from '../../skeletons/ListSkeleton';
import RunListItem from './RunListItem';
import { RunListFragment_runConnection$key } from './__generated__/RunListFragment_runConnection.graphql';

interface Props {
    fragmentRef: RunListFragment_runConnection$key
    hasNext: boolean
    loadNext: (count: number) => void
    displayWorkspacePath?: boolean
}

function RunList({ fragmentRef, hasNext, loadNext, displayWorkspacePath }: Props) {
    const theme = useTheme();
    const mobile = useMediaQuery(theme.breakpoints.down('lg'))

    const data = useFragment<RunListFragment_runConnection$key>(
        graphql`
      fragment RunListFragment_runConnection on RunConnection {
        totalCount
        edges {
            node {
                id
                ...RunListItemFragment_run
            }
        }
      }
    `, fragmentRef);

    const edgeCount = data.edges?.length ?? 0;

    return (
        <InfiniteScroll
            dataLength={edgeCount}
            next={() => loadNext(20)}
            hasMore={hasNext}
            loader={<ListSkeleton rowCount={3} />}
        >

            {!mobile && <TableContainer>
                <Table sx={{ minWidth: 650, tableLayout: 'fixed' }} aria-label="workspace runs">
                    <TableHead>
                        <TableRow>
                            <TableCell>Status</TableCell>
                            <TableCell>Run ID</TableCell>
                            {displayWorkspacePath && <TableCell width={200}>Workspace</TableCell>}
                            <TableCell>Type</TableCell>
                            <TableCell>Triggerer</TableCell>
                            <TableCell>Stages</TableCell>
                            <TableCell width={100}></TableCell>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {data.edges?.map((edge: any) => (
                            <RunListItem
                                key={edge.node.id}
                                runFragment={edge.node}
                                displayWorkspacePath={displayWorkspacePath}
                                mobile={mobile}
                            />
                        ))}
                    </TableBody>
                </Table>
            </TableContainer>}
            {mobile && <List>
                {data.edges?.map((edge: any, index: number) => (
                    <RunListItem
                        key={edge.node.id}
                        runFragment={edge.node}
                        displayWorkspacePath={displayWorkspacePath}
                        mobile={mobile}
                        last={index === edgeCount - 1}
                    />
                ))}
            </List>}
        </InfiniteScroll>
    );
}

export default RunList;
