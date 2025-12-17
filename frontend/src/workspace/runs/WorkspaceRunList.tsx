import { Box, Typography } from '@mui/material';
import Paper from '@mui/material/Paper';
import graphql from 'babel-plugin-relay/macro';
import { useLazyLoadQuery, usePaginationFragment } from "react-relay/hooks";
import { ConnectionHandler } from 'relay-runtime';
import { WorkspaceRunListFragment_runs$key } from './__generated__/WorkspaceRunListFragment_runs.graphql';
import { WorkspaceRunListPaginationQuery } from './__generated__/WorkspaceRunListPaginationQuery.graphql';
import { WorkspaceRunListQuery } from './__generated__/WorkspaceRunListQuery.graphql';
import RunList from './RunList';

const INITIAL_ITEM_COUNT = 50;

interface Props {
    workspaceId: string
    includeAssessmentRuns: boolean
}

export function GetConnections(workspaceId: string): string[] {
    return [
        { workspaceId, sort: 'CREATED_AT_DESC', workspaceAssessment: false },
        { workspaceId, sort: 'CREATED_AT_DESC', workspaceAssessment: null }
    ].map(vars => ConnectionHandler.getConnectionID(
        "root",
        'WorkspaceRunList_runs',
        vars
    ));
}

function WorkspaceRunList({ workspaceId, includeAssessmentRuns }: Props) {

    const queryData = useLazyLoadQuery<WorkspaceRunListQuery>(graphql`
        query WorkspaceRunListQuery($first: Int, $last: Int, $after: String, $before: String, $workspaceId: String, $workspaceAssessment: Boolean) {
            ...WorkspaceRunListFragment_runs
        }
    `, {
        first: INITIAL_ITEM_COUNT,
        workspaceId,
        workspaceAssessment: includeAssessmentRuns ? null : false
    }, { fetchPolicy: 'network-only' })

    const { data, loadNext, hasNext } = usePaginationFragment<WorkspaceRunListPaginationQuery, WorkspaceRunListFragment_runs$key>(
        graphql`
      fragment WorkspaceRunListFragment_runs on Query
      @refetchable(queryName: "WorkspaceRunListPaginationQuery") {
        runs(
            after: $after
            before: $before
            first: $first
            last: $last
            workspaceId: $workspaceId
            sort: CREATED_AT_DESC
            workspaceAssessment: $workspaceAssessment
        ) @connection(key: "WorkspaceRunList_runs") {
            totalCount
            edges {
                node {
                    id
                }
            }
            ...RunListFragment_runConnection
        }
      }
    `, queryData);

    return (
        <Box>
            {data.runs.edges && data.runs.edges.length > 0 && <RunList fragmentRef={data.runs} hasNext={hasNext} loadNext={loadNext} />}
            {data.runs.edges?.length === 0 && <Paper variant="outlined" sx={{ marginTop: 4, display: 'flex', justifyContent: 'center' }}>
                <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center">
                    <Typography variant="h6" color="textSecondary" align="center">
                        No runs have been created in this workspace
                    </Typography>
                </Box>
            </Paper>}
        </Box>
    );
}

export default WorkspaceRunList;
