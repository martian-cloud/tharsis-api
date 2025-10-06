import { useMemo, useState } from 'react';
import { Box, Link, List, ListItem, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useLazyLoadQuery, usePaginationFragment, useSubscription } from 'react-relay/hooks';
import HomeRunListItem from './HomeRunListItem';
import { ConnectionHandler, ConnectionInterface, GraphQLSubscriptionConfig, RecordSourceProxy } from 'relay-runtime';
import { HomeRunListSubscription, HomeRunListSubscription$data } from './__generated__/HomeRunListSubscription.graphql';
import { HomeRunListFragment_runs$key } from './__generated__/HomeRunListFragment_runs.graphql';
import { HomeRunListPaginationQuery } from './__generated__/HomeRunListPaginationQuery.graphql';
import { HomeRunListQuery } from './__generated__/HomeRunListQuery.graphql';

const INITIAL_ITEM_COUNT = 5;

function GetConnections(): [string] {
    const connectionId = ConnectionHandler.getConnectionID(
        "root",
        "HomeRunList_runs",
        { sort: 'UPDATED_AT_DESC', workspaceAssessment: false }
    );
    return [connectionId];
}

const query = graphql`
    query HomeRunListQuery($first: Int!, $after: String) {
        ...HomeRunListFragment_runs
    }
`;

const runSubscription = graphql` subscription HomeRunListSubscription($input: RunSubscriptionInput!) {
    workspaceRunEvents(input: $input) {
        action
        run {
            id
            ...HomeRunListItemFragment_run
        }
    }
}`;

function HomeRunList() {
    const theme = useTheme();
    const [displayCount, setDisplayCount] = useState<number>(INITIAL_ITEM_COUNT);
    const queryData = useLazyLoadQuery<HomeRunListQuery>(query, { first: INITIAL_ITEM_COUNT }, { fetchPolicy: 'store-and-network' });

    const { data, loadNext, hasNext } = usePaginationFragment<HomeRunListPaginationQuery, HomeRunListFragment_runs$key>(
        graphql`
        fragment HomeRunListFragment_runs on Query
        @refetchable(queryName: "HomeRunListPaginationQuery") {
            runs(
                first: $first
                after: $after
                sort: UPDATED_AT_DESC
                workspaceAssessment: false
                ) @connection(key: "HomeRunList_runs") {
                    totalCount
                    edges {
                        node {
                            id
                            ...HomeRunListItemFragment_run
                        }
                    }
                }
            }
        `, queryData
    );

    const runSubscriptionConfig = useMemo<GraphQLSubscriptionConfig<HomeRunListSubscription>>(() => ({
        subscription: runSubscription,
        variables: { input: {} },
        onCompleted: () => console.log("Subscription completed"),
        onError: (error) => console.warn(`Subscription error: ${error.message}`),
        updater: (store: RecordSourceProxy, payload: HomeRunListSubscription$data | null | undefined) => {
            if (!payload) {
                return;
            }
            const record = store.get(payload.workspaceRunEvents.run.id);
            if (record == null) {
                return;
            }
            GetConnections().forEach(id => {
                const connectionRecord = store.get(id);
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
    }), []);

    useSubscription(runSubscriptionConfig);

    const edges = data?.runs?.edges ?? [];
    const displayedEdges = edges.slice(0, displayCount);
    const edgeCount = displayedEdges.length;

    return (
        <Box>
            <Typography variant="subtitle1" fontWeight={600}>Recent Runs</Typography>
            {edgeCount === 0 && <Box sx={{ mt: 2, mb: 2, p: 2, border: `1px dashed ${theme.palette.divider}`, borderRadius: 2 }}>
                <Typography color="textSecondary" variant="body2">
                    There are no recent runs.
                </Typography>
            </Box>}
            {edgeCount > 0 && <List>
                {displayedEdges.map((edge: any, index: number) => <HomeRunListItem
                    key={edge.node.id}
                    last={index === (edgeCount - 1)}
                    fragmentRef={edge.node} />
                )}
                {(hasNext || edges.length > displayCount) && <ListItem>
                    <Link
                        variant="body2"
                        color="textSecondary"
                        sx={{ cursor: 'pointer' }}
                        underline="hover"
                        onClick={() => {
                            if (edges.length < displayCount + INITIAL_ITEM_COUNT) {
                                loadNext(INITIAL_ITEM_COUNT);
                            }
                            setDisplayCount(displayCount + INITIAL_ITEM_COUNT);
                        }}
                    >
                        Show more
                    </Link>
                </ListItem>}
            </List>}
        </Box>
    );
}

export default HomeRunList;
