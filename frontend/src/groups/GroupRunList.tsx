import { Box, Paper, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useMemo } from 'react';
import { useLazyLoadQuery, usePaginationFragment, useSubscription } from "react-relay/hooks";
import { ConnectionHandler, ConnectionInterface, GraphQLSubscriptionConfig, RecordSourceProxy } from 'relay-runtime';
import RunList from '../workspace/runs/RunList';
import { GroupRunListFragment_group$key } from './__generated__/GroupRunListFragment_group.graphql';
import { GroupRunListPaginationQuery } from './__generated__/GroupRunListPaginationQuery.graphql';
import { GroupRunListQuery } from './__generated__/GroupRunListQuery.graphql';
import { GroupRunListSubscription, GroupRunListSubscription$data } from './__generated__/GroupRunListSubscription.graphql';

const INITIAL_ITEM_COUNT = 50;

interface Props {
    groupPath: string
    includeAssessmentRuns: boolean
}

function GetConnections(includeAssessmentRuns: boolean): string[] {
    const connectionId = ConnectionHandler.getConnectionID(
        'root',
        'GroupRunList_runs',
        { sort: 'CREATED_AT_DESC', workspaceAssessment: includeAssessmentRuns ? null : false, includeNestedRuns: true }
    );
    return [connectionId];
}

const runSubscription = graphql`
    subscription GroupRunListSubscription($input: RunSubscriptionInput!) {
        workspaceRunEvents(input: $input) {
            action
            run {
                id
                assessment
                ...RunListItemFragment_run
            }
        }
    }
`;

function GroupRunList({ groupPath, includeAssessmentRuns }: Props) {

    const queryData = useLazyLoadQuery<GroupRunListQuery>(graphql`
        query GroupRunListQuery($first: Int, $after: String, $groupPath: String!, $workspaceAssessment: Boolean, $includeNestedRuns: Boolean) {
            ...GroupRunListFragment_group
        }
    `, {
        first: INITIAL_ITEM_COUNT,
        groupPath,
        workspaceAssessment: includeAssessmentRuns ? null : false,
        includeNestedRuns: true
    }, { fetchPolicy: 'store-and-network' })

    const { data, loadNext, hasNext } = usePaginationFragment<GroupRunListPaginationQuery, GroupRunListFragment_group$key>(
        graphql`
            fragment GroupRunListFragment_group on Query
            @refetchable(queryName: "GroupRunListPaginationQuery") {
                group(fullPath: $groupPath) {
                    id
                    runs(
                        first: $first
                        after: $after
                        sort: CREATED_AT_DESC
                        workspaceAssessment: $workspaceAssessment
                        includeNestedRuns: $includeNestedRuns
                    ) @connection(key: "GroupRunList_runs") {
                        totalCount
                        edges {
                            node {
                                id
                            }
                        }
                        ...RunListFragment_runConnection
                    }
                }
            }
        `, queryData
    );

    const runSubscriptionConfig = useMemo<GraphQLSubscriptionConfig<GroupRunListSubscription>>(() => ({
        subscription: runSubscription,
        variables: { input: { ancestorGroupId: data.group?.id } },
        onCompleted: () => console.log("Group run subscription completed"),
        onError: (error) => console.warn(`Group run subscription error: ${error.message}`),
        updater: (store: RecordSourceProxy, payload: GroupRunListSubscription$data | null | undefined) => {
            if (!payload) {
                return;
            }
            if (payload.workspaceRunEvents.run.assessment && !includeAssessmentRuns) {
                return;
            }
            const record = store.get(payload.workspaceRunEvents.run.id);
            if (record == null) {
                return;
            }
            GetConnections(includeAssessmentRuns).forEach(id => {
                let connectionRecord = store.get(id);
                if (!connectionRecord) {
                    const groupRecord = store.getRoot().getLinkedRecord('group', { fullPath: groupPath });
                    if (groupRecord) {
                        connectionRecord = ConnectionHandler.getConnection(groupRecord, 'GroupRunList_runs', {
                            sort: 'CREATED_AT_DESC',
                            workspaceAssessment: includeAssessmentRuns ? null : false,
                            includeNestedRuns: true
                        });
                    }
                }
                if (connectionRecord) {
                    const { NODE, EDGES } = ConnectionInterface.get();

                    const recordId = record.getDataID();
                    // Check if edge already exists in connection
                    const nodeAlreadyExistsInConnection = connectionRecord
                        .getLinkedRecords(EDGES)
                        ?.some(
                            edge => edge?.getLinkedRecord(NODE)?.getDataID() === recordId,
                        );
                    if (!nodeAlreadyExistsInConnection) {
                        // Create Edge
                        const edge = ConnectionHandler.createEdge(
                            store,
                            connectionRecord,
                            record,
                            'RunEdge'
                        );
                        if (edge) {
                            // Add edge to the beginning of the connection
                            ConnectionHandler.insertEdgeBefore(
                                connectionRecord,
                                edge,
                            );
                        }
                    }
                }
            });
        }
    }), [data.group?.id]);

    useSubscription(runSubscriptionConfig);

    const runs = data.group?.runs?.edges || [];

    return (
        <Box>
            {data.group?.runs && runs.length > 0 && <RunList fragmentRef={data.group?.runs} hasNext={hasNext} loadNext={loadNext} displayWorkspacePath />}
            {runs.length === 0 && <Paper variant="outlined" sx={{ marginTop: 4, display: 'flex', justifyContent: 'center' }}>
                <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center">
                    <Typography variant="h6" color="textSecondary" align="center">
                        No runs have been created in this group or its subgroups
                    </Typography>
                </Box>
            </Paper>}
        </Box>
    );
}

export default GroupRunList;
