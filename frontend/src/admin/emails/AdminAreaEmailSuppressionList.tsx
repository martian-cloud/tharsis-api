import { Box, CircularProgress, Paper, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import throttle from 'lodash.throttle';
import { Suspense, useEffect, useMemo, useState } from 'react';
import InfiniteScroll from 'react-infinite-scroll-component';
import { fetchQuery, useLazyLoadQuery, usePaginationFragment, useRelayEnvironment } from 'react-relay/hooks';
import SearchInput from '../../common/SearchInput';
import { ResponsiveTable } from '../../common/ResponsiveTable';
import ListSkeleton from '../../skeletons/ListSkeleton';
import AdminAreaBreadcrumbs from '../AdminAreaBreadcrumbs';
import AdminAreaEmailSuppressionListItem from './AdminAreaEmailSuppressionListItem';
import { AdminAreaEmailSuppressionListFragment_suppressions$key } from './__generated__/AdminAreaEmailSuppressionListFragment_suppressions.graphql';
import { AdminAreaEmailSuppressionListPaginationQuery } from './__generated__/AdminAreaEmailSuppressionListPaginationQuery.graphql';
import { AdminAreaEmailSuppressionListQuery } from './__generated__/AdminAreaEmailSuppressionListQuery.graphql';

const DESCRIPTION = 'Email addresses the system will not send to. Addresses are suppressed automatically after a hard bounce or a spam complaint.';
const SUPPRESSIONS_INITIAL_ITEM_COUNT = 20;

const BREADCRUMB_ROUTES = [{ title: 'email suppressions', path: 'email_suppressions' }];

const query = graphql`
    query AdminAreaEmailSuppressionListQuery($first: Int, $after: String, $search: String) {
        ...AdminAreaEmailSuppressionListFragment_suppressions
    }
`;

const COLUMNS = [
    { label: 'Address' },
    { label: 'Cause' },
    { label: 'Created' },
    { label: '', align: 'right' as const },
];

function AdminAreaEmailSuppressionListContent() {
    const theme = useTheme();
    const environment = useRelayEnvironment();
    const [search, setSearch] = useState('');
    const [isRefreshing, setIsRefreshing] = useState(false);

    const queryData = useLazyLoadQuery<AdminAreaEmailSuppressionListQuery>(
        query,
        { first: SUPPRESSIONS_INITIAL_ITEM_COUNT },
        { fetchPolicy: 'store-and-network' }
    );

    const { data, loadNext, hasNext, refetch } = usePaginationFragment<
        AdminAreaEmailSuppressionListPaginationQuery,
        AdminAreaEmailSuppressionListFragment_suppressions$key
    >(
        graphql`
            fragment AdminAreaEmailSuppressionListFragment_suppressions on Query
            @refetchable(queryName: "AdminAreaEmailSuppressionListPaginationQuery") {
                emailSuppressions(first: $first, after: $after, search: $search, sort: CREATED_AT_DESC)
                @connection(key: "AdminAreaEmailSuppressionList_emailSuppressions") {
                    __id
                    totalCount
                    edges {
                        node {
                            id
                            ...AdminAreaEmailSuppressionListItemFragment_suppression
                        }
                    }
                }
            }
        `,
        queryData
    );

    const fetch = useMemo(
        () =>
            throttle(
                (input: string) => {
                    setIsRefreshing(true);
                    const normalizedInput = input.trim();
                    fetchQuery(environment, query, { first: SUPPRESSIONS_INITIAL_ITEM_COUNT, search: normalizedInput })
                        .subscribe({
                            complete: () => {
                                setIsRefreshing(false);
                                setSearch(input);
                                refetch(
                                    { first: SUPPRESSIONS_INITIAL_ITEM_COUNT, search: normalizedInput },
                                    { fetchPolicy: 'store-only' }
                                );
                            },
                            error: () => setIsRefreshing(false),
                        });
                },
                2000,
                { leading: false, trailing: true }
            ),
        [environment, refetch]
    );

    useEffect(() => () => { fetch.cancel(); }, [fetch]);

    const onSearchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        fetch(event.target.value);
    };

    const onSearchKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
        if (event.key === 'Enter') {
            fetch.flush();
        }
    };

    const suppressions = (data.emailSuppressions?.edges ?? [])
        .map(edge => edge?.node)
        .filter((node): node is NonNullable<typeof node> => node != null);
    const totalCount = data.emailSuppressions?.totalCount ?? 0;

    const connectionId = data.emailSuppressions?.__id;

    return (
        <Box>
            <AdminAreaBreadcrumbs childRoutes={BREADCRUMB_ROUTES} />
            <Typography variant="h5" gutterBottom>Email Suppressions</Typography>
            <Typography variant="body2" sx={{ mb: 2 }}>{DESCRIPTION}</Typography>

            <Box marginBottom={2}>
                <SearchInput
                    fullWidth
                    placeholder="Search by address"
                    onChange={onSearchChange}
                    onKeyDown={onSearchKeyDown}
                />
            </Box>

            <Paper variant="outlined" sx={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0 }}>
                <Box padding={2}>
                    <Typography variant="subtitle1">
                        {totalCount} {totalCount === 1 ? 'suppressed address' : 'suppressed addresses'}
                    </Typography>
                </Box>
            </Paper>

            <Box sx={{
                borderBottom: `1px solid ${theme.palette.divider}`,
                borderLeft: `1px solid ${theme.palette.divider}`,
                borderRight: `1px solid ${theme.palette.divider}`,
                borderBottomLeftRadius: 4,
                borderBottomRightRadius: 4,
            }}>
                {suppressions.length === 0 ? (
                    <Typography align="center" color="textSecondary" padding={4}>
                        {search ? <>No suppressed addresses matching <strong>{search}</strong></> : 'No suppressed addresses'}
                    </Typography>
                ) : (
                    <InfiniteScroll
                        dataLength={suppressions.length}
                        next={() => loadNext(20)}
                        hasMore={hasNext}
                        loader={<ListSkeleton rowCount={3} />}
                    >
                        <Box sx={isRefreshing ? { opacity: 0.5 } : undefined}>
                            <ResponsiveTable ariaLabel="email suppressions" columns={COLUMNS}>
                                {suppressions.map(suppression => (
                                    <AdminAreaEmailSuppressionListItem key={suppression.id} fragmentRef={suppression} connectionId={connectionId} />
                                ))}
                            </ResponsiveTable>
                        </Box>
                    </InfiniteScroll>
                )}
            </Box>
        </Box>
    );
}

function AdminAreaEmailSuppressionList() {
    return (
        <Suspense
            fallback={
                <Box
                    sx={{
                        width: '100%',
                        height: `calc(100vh - 64px)`,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center'
                    }}
                >
                    <CircularProgress />
                </Box>
            }
        >
            <AdminAreaEmailSuppressionListContent />
        </Suspense>
    );
}

export default AdminAreaEmailSuppressionList;
