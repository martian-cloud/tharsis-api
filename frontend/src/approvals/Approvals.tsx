import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import { Alert, Box, Link, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { Fragment, Suspense, useState } from 'react';
import InfiniteScroll from 'react-infinite-scroll-component';
import { useLazyLoadQuery, usePaginationFragment } from 'react-relay/hooks';
import { Link as RouterLink } from 'react-router-dom';
import { MutationError } from '../common/error';
import { PageLayoutProvider } from '../layout/PageLayoutContext';
import ListSkeleton from '../skeletons/ListSkeleton';
import ApprovalGateCard from './ApprovalGateCard';
import { ApprovalsFragment_gates$key } from './__generated__/ApprovalsFragment_gates.graphql';
import { ApprovalsPaginationQuery } from './__generated__/ApprovalsPaginationQuery.graphql';
import { ApprovalsQuery } from './__generated__/ApprovalsQuery.graphql';

const INITIAL_ITEM_COUNT = 20;
const DESCRIPTION = 'Run policies that need your review before the run can proceed';

const query = graphql`
    query ApprovalsQuery($first: Int, $after: String) {
        ...ApprovalsFragment_gates
    }
`;

function ApprovalList() {
    const theme = useTheme();
    const [error, setError] = useState<MutationError>();
    // Decided gates are removed from the connection by @deleteEdge, which leaves totalCount alone —
    // count them here so the header doesn't sit one ahead of the cards below it.
    const [decidedCount, setDecidedCount] = useState(0);

    const queryData = useLazyLoadQuery<ApprovalsQuery>(query, { first: INITIAL_ITEM_COUNT }, { fetchPolicy: 'store-and-network' });

    const { data, loadNext, hasNext } = usePaginationFragment<ApprovalsPaginationQuery, ApprovalsFragment_gates$key>(
        graphql`
        fragment ApprovalsFragment_gates on Query
        @refetchable(queryName: "ApprovalsPaginationQuery") {
            runGatesAwaitingMyDecision(
                after: $after
                first: $first
                sort: CREATED_AT_DESC
            ) @connection(key: "Approvals_runGatesAwaitingMyDecision") {
                # The connection's own store id, which is what the decision buttons' @deleteEdge
                # needs. Reading it here means the id keeps matching whatever arguments identify
                # the connection, instead of us re-deriving it from the key and having to keep
                # those arguments out of its identity.
                __id
                totalCount
                edges {
                    node {
                        id
                        ...ApprovalGateCardFragment_gate
                    }
                }
            }
        }
    `, queryData);

    const connection = data.runGatesAwaitingMyDecision;
    const connectionIds = connection ? [connection.__id] : [];
    const edges = connection?.edges ?? [];
    const count = Math.max((connection?.totalCount ?? 0) - decidedCount, 0);

    return (
        <Box>
            <Link
                component={RouterLink}
                to="/"
                underline="hover"
                variant="body2"
                sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.875, mb: 2 }}
            >
                <ArrowBackIcon sx={{ width: 16, height: 16 }} />
                Back to dashboard
            </Link>

            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, mb: 0.75 }}>
                <Typography variant="h5">Policies Awaiting My Approval</Typography>
            </Box>
            <Typography variant="body2" color="textSecondary" sx={{ mb: 2, maxWidth: 640 }}>
                {DESCRIPTION}
            </Typography>

            {error && <Alert sx={{ mb: 2 }} severity={error.severity}>{error.message}</Alert>}

            {edges.length === 0 ? (
                <Typography
                    align="center"
                    color="textSecondary"
                    sx={{ padding: 4, border: `1px solid ${theme.palette.divider}`, borderRadius: '8px' }}
                >
                    No approvals are waiting on you
                </Typography>
            ) : (
                <Fragment>
                    <Typography variant="subtitle1" color="textSecondary" mb={1}>
                        {count} pending approval request{count === 1 ? '' : 's'}
                    </Typography>
                    <InfiniteScroll
                        dataLength={edges.length}
                        next={() => loadNext(INITIAL_ITEM_COUNT)}
                        hasMore={hasNext}
                        loader={<ListSkeleton rowCount={3} />}
                    >
                        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                            {edges.map(edge => edge?.node && (
                                <ApprovalGateCard
                                    key={edge.node.id}
                                    fragmentRef={edge.node}
                                    connectionIds={connectionIds}
                                    onDecided={() => setDecidedCount(decided => decided + 1)}
                                    onError={setError}
                                />
                            ))}
                        </Box>
                    </InfiniteScroll>
                </Fragment>
            )
            }
        </Box >
    );
}

function Approvals() {
    return (
        <PageLayoutProvider>
            <Suspense fallback={<ListSkeleton rowCount={5} size="large" />}>
                <ApprovalList />
            </Suspense>
        </PageLayoutProvider>
    );
}

export default Approvals;
