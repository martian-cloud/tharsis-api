import { Box, Button, List, Paper, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useState } from 'react';
import InfiniteScroll from 'react-infinite-scroll-component';
import { ConnectionHandler, useFragment, useLazyLoadQuery, useMutation, usePaginationFragment } from 'react-relay/hooks';
import { MutationError } from '../common/error';
import { ServiceAccountOption } from '../namespace/members/ServiceAccountAutocomplete';
import ListSkeleton from '../skeletons/ListSkeleton';
import AssignServiceAccountDialog from './AssignServiceAccountDialog';
import AssignedServiceAccountListItem from './AssignedServiceAccountListItem';
import UnassignServiceAccountDialog from './UnassignServiceAccountDialog';
import { AssignedServiceAccountListAssignMutation } from './__generated__/AssignedServiceAccountListAssignMutation.graphql';
import { AssignedServiceAccountListFragment_assignedServiceAccounts$key } from './__generated__/AssignedServiceAccountListFragment_assignedServiceAccounts.graphql';
import { AssignedServiceAccountListFragment_runner$key } from './__generated__/AssignedServiceAccountListFragment_runner.graphql';
import { AssignedServiceAccountListPaginationQuery } from './__generated__/AssignedServiceAccountListPaginationQuery.graphql';
import { AssignedServiceAccountListQuery } from './__generated__/AssignedServiceAccountListQuery.graphql';
import { AssignedServiceAccountListUnassignMutation } from './__generated__/AssignedServiceAccountListUnassignMutation.graphql';

function GetConnections(runnerId: string): [string] {
    const connectionId = ConnectionHandler.getConnectionID(
        runnerId,
        "AssignedServiceAccountList_assignedServiceAccounts"
    );
    return [connectionId];
}

const query = graphql`
    query AssignedServiceAccountListQuery($id: String!, $first: Int!, $after: String) {
        node(id: $id) {
            ... on Runner {
                id
                ...AssignedServiceAccountListFragment_assignedServiceAccounts
            }
        }
    }`;

interface Props {
    fragmentRef: AssignedServiceAccountListFragment_runner$key;
}

function AssignedServiceAccountList({ fragmentRef }: Props) {
    const [showAssignServiceAccountDialog, setShowAssignServiceAccountDialog] = useState(false);
    const [error, setError] = useState<MutationError | null>(null);
    const [serviceAccountToUnassignPath, setServiceAccountToUnassignPath] = useState<string | null>(null);
    const theme = useTheme();

    const runner = useFragment<AssignedServiceAccountListFragment_runner$key>(graphql`
        fragment AssignedServiceAccountListFragment_runner on Runner
        {
            id
            resourcePath
        }
    `, fragmentRef);

    const queryData = useLazyLoadQuery<AssignedServiceAccountListQuery>(query, { id: runner.id, first: 100 }, { fetchPolicy: 'store-and-network' });

    const { data, loadNext, hasNext } = usePaginationFragment<AssignedServiceAccountListPaginationQuery, AssignedServiceAccountListFragment_assignedServiceAccounts$key>(
        graphql`
        fragment AssignedServiceAccountListFragment_assignedServiceAccounts on Runner
        @refetchable(queryName: "AssignedServiceAccountListPaginationQuery") {
                type
                resourcePath
                group {
                    fullPath
                }
                assignedServiceAccounts(
                    first: $first
                    after: $after
                ) @connection(key: "AssignedServiceAccountList_assignedServiceAccounts") {
                    totalCount
                    edges {
                        node {
                            id
                            ...AssignedServiceAccountListItemFragment_assignedServiceAccount
                        }
                    }
                }
            }
        `, queryData.node
    );

    const [commitAssign, assignCommitInFlight] = useMutation<AssignedServiceAccountListAssignMutation>(graphql`
        mutation AssignedServiceAccountListAssignMutation($input: AssignServiceAccountToRunnerInput!, $connections: [ID!]!) {
            assignServiceAccountToRunner(input: $input) {
                runner {
                    assignedServiceAccounts (first: 0) {
                        totalCount
                    }
                }
                serviceAccount @prependNode(connections: $connections, edgeTypeName: "ServiceAccountEdge") {
                    id
                    ...AssignedServiceAccountListItemFragment_assignedServiceAccount
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const resetAssignDialog = () => {
        setShowAssignServiceAccountDialog(false)
        setServiceAccountToUnassignPath(null)
        setError(null)
    };

    const resetUnassignDialog = () => {
        setServiceAccountToUnassignPath(null)
        setError(null)
    };

    const assignServiceAccount = (serviceAccount?: ServiceAccountOption) => {
        if (serviceAccount) {
            commitAssign({
                variables: {
                    input: {
                        runnerPath: runner.resourcePath,
                        serviceAccountPath: serviceAccount.resourcePath
                    },
                    connections: GetConnections(runner.id)
                },
                updater: (store, data) => {
                    if (data && data.assignServiceAccountToRunner.serviceAccount) {
                        const runnerRecord = store.get(runner.id);
                        if (runnerRecord) {
                            const connectionRecord = ConnectionHandler.getConnection(
                                runnerRecord,
                                'AssignedServiceAccountList_assignedServiceAccounts',
                            );
                            if (connectionRecord) {
                                const totalCount = connectionRecord.getValue('totalCount') as number;
                                connectionRecord.setValue(totalCount + 1, 'totalCount');
                            }
                        }
                    }
                },
                onCompleted: data => {
                    if (data && data.assignServiceAccountToRunner.problems.length) {
                        setError({
                            severity: 'warning',
                            message: data.assignServiceAccountToRunner.problems.map(problem => problem.message).join('; ')
                        });
                    } else {
                        resetAssignDialog()
                    }
                },
                onError: error => {
                    setError({
                        severity: 'error',
                        message: `Unexpected Error Occurred: ${error.message}`
                    });
                }
            })
        } else {
            resetAssignDialog()
        }
    };

    const [commitUnassign, unAssignCommitInFlight] = useMutation<AssignedServiceAccountListUnassignMutation>(graphql`
        mutation AssignedServiceAccountListUnassignMutation($input: AssignServiceAccountToRunnerInput!, $connections: [ID!]!) {
            unassignServiceAccountFromRunner(input: $input) {
                runner {
                    assignedServiceAccounts (first: 0) {
                        totalCount
                    }
                }
                serviceAccount {
                    id @deleteEdge(connections: $connections)
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const unassignServiceAccount = (confirm?: boolean) => {
        if (confirm && serviceAccountToUnassignPath) {
            commitUnassign({
                variables: {
                    input: {
                        runnerPath: runner.resourcePath,
                        serviceAccountPath: serviceAccountToUnassignPath
                    },
                    connections: GetConnections(runner.id)
                },
                updater: (store, data) => {
                    if (data && data.unassignServiceAccountFromRunner.serviceAccount) {
                        const runnerRecord = store.get(runner.id);
                        if (runnerRecord) {
                            const connectionRecord = ConnectionHandler.getConnection(
                                runnerRecord,
                                'AssignedServiceAccountList_assignedServiceAccounts',
                            );
                            if (connectionRecord) {
                                const totalCount = connectionRecord.getValue('totalCount') as number;
                                connectionRecord.setValue(totalCount - 1, 'totalCount');
                            }
                        }
                    }
                },
                onCompleted: data => {
                    if (data && data.unassignServiceAccountFromRunner.problems.length) {
                        setError({
                            severity: 'warning',
                            message: data.unassignServiceAccountFromRunner.problems.map(problem => problem.message).join('; ')
                        });
                    }
                    else {
                        resetUnassignDialog()
                    }
                },
                onError: error => {
                    setError({
                        severity: 'error',
                        message: `Unexpected Error Occurred: ${error.message}`
                    });
                }
            })
        } else {
            resetUnassignDialog()
        }
    };

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
                <Typography color="textSecondary">A service account is used by the runner to authenticate with the Tharsis API.</Typography>
                <Button
                    sx={{ width: 225 }}
                    size="small"
                    color="secondary"
                    variant="outlined"
                    onClick={() => setShowAssignServiceAccountDialog(true)}
                >
                    Assign Service Account
                </Button>
            </Box>
            {(!data?.assignedServiceAccounts?.edges || data?.assignedServiceAccounts?.edges?.length === 0) ? <Paper sx={{ p: 2, m: 2 }}>
                <Typography>No service accounts are assigned to this runner.</Typography>
            </Paper>
                :
                <Box sx={{ p: 2 }}>
                    <Paper sx={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0, border: `1px solid ${theme.palette.divider}` }}>
                        <Box padding={2}>
                            <Typography variant="subtitle1">
                                {data?.assignedServiceAccounts.totalCount} assigned service account{data?.assignedServiceAccounts.totalCount !== 1 && 's'}
                            </Typography>
                        </Box>
                    </Paper>
                    <InfiniteScroll
                        dataLength={(data?.assignedServiceAccounts?.edges && data?.assignedServiceAccounts?.edges.length) ?? 0}
                        next={() => loadNext(20)}
                        hasMore={hasNext}
                        loader={<ListSkeleton rowCount={3} />}
                    >
                        <List disablePadding>{data?.assignedServiceAccounts.edges?.map((edge: any) => <AssignedServiceAccountListItem
                            key={edge.node.id}
                            fragmentRef={edge.node}
                            onDelete={setServiceAccountToUnassignPath}
                        />)}
                        </List>
                    </InfiniteScroll>
                </Box>}
            {showAssignServiceAccountDialog && data?.group && <AssignServiceAccountDialog
                error={error}
                namespacePath={data?.group.fullPath}
                onClose={assignServiceAccount}
                assignCommitInFlight={assignCommitInFlight}
            />}
            {serviceAccountToUnassignPath && <UnassignServiceAccountDialog
                error={error}
                onClose={unassignServiceAccount}
                name={serviceAccountToUnassignPath}
                unAssignCommitInFlight={unAssignCommitInFlight}
            />}
        </Box>
    );
}

export default AssignedServiceAccountList
