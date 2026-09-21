import FilterListIcon from '@mui/icons-material/FilterList';
import { Box, Checkbox, CircularProgress, FormControlLabel, IconButton, Menu, Paper, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import throttle from 'lodash.throttle';
import { Suspense, useEffect, useMemo, useState } from 'react';
import InfiniteScroll from 'react-infinite-scroll-component';
import { fetchQuery, useLazyLoadQuery, usePaginationFragment, useRelayEnvironment } from 'react-relay/hooks';
import SearchInput from '../../common/SearchInput';
import { ResponsiveTable } from '../../common/ResponsiveTable';
import ListSkeleton from '../../skeletons/ListSkeleton';
import AdminAreaBreadcrumbs from '../AdminAreaBreadcrumbs';
import AdminAreaEmailOutboxListItem from './AdminAreaEmailOutboxListItem';
import { AdminAreaEmailOutboxListFragment_outboxes$key } from './__generated__/AdminAreaEmailOutboxListFragment_outboxes.graphql';
import { AdminAreaEmailOutboxListPaginationQuery } from './__generated__/AdminAreaEmailOutboxListPaginationQuery.graphql';
import { AdminAreaEmailOutboxListQuery } from './__generated__/AdminAreaEmailOutboxListQuery.graphql';

const DESCRIPTION = 'Emails queued or sent by the system, each fanned out to one or more recipients. Select an email to view its recipients and delivery status.';
const OUTBOX_INITIAL_ITEM_COUNT = 20;

const BREADCRUMB_ROUTES = [{ title: 'email outbox', path: 'email_outbox' }];

const query = graphql`
    query AdminAreaEmailOutboxListQuery($first: Int, $after: String, $subjectSearch: String, $ephemeral: Boolean) {
        ...AdminAreaEmailOutboxListFragment_outboxes
    }
`;

const COLUMNS = [
    { label: 'Subject' },
    { label: 'Type' },
    { label: 'Lifecycle' },
    { label: 'Created', align: 'right' as const },
];

function AdminAreaEmailOutboxListContent() {
    const theme = useTheme();
    const environment = useRelayEnvironment();
    const [search, setSearch] = useState('');
    const [hideEphemeral, setHideEphemeral] = useState(false);
    const [isRefreshing, setIsRefreshing] = useState(false);
    const [menuAnchorEl, setMenuAnchorEl] = useState<Element | null>(null);

    // Default to showing everything; "Hide ephemeral" narrows to non-ephemeral only.
    const ephemeralFilter = hideEphemeral ? false : undefined;

    // Ephemeral is intentionally left out of the initial query variables (matching the
    // default filter) so toggling it never changes the top-level query's variables and
    // re-suspends the whole list; the toggle is applied via the same prefetch-then-refetch
    // flow as search, below.
    const queryData = useLazyLoadQuery<AdminAreaEmailOutboxListQuery>(
        query,
        { first: OUTBOX_INITIAL_ITEM_COUNT },
        { fetchPolicy: 'store-and-network' }
    );

    const { data, loadNext, hasNext, refetch } = usePaginationFragment<
        AdminAreaEmailOutboxListPaginationQuery,
        AdminAreaEmailOutboxListFragment_outboxes$key
    >(
        graphql`
            fragment AdminAreaEmailOutboxListFragment_outboxes on Query
            @refetchable(queryName: "AdminAreaEmailOutboxListPaginationQuery") {
                emailOutboxItems(first: $first, after: $after, subjectSearch: $subjectSearch, ephemeral: $ephemeral, sort: CREATED_AT_DESC)
                @connection(key: "AdminAreaEmailOutboxList_emailOutboxItems") {
                    totalCount
                    edges {
                        node {
                            id
                            ...AdminAreaEmailOutboxListItemFragment_outbox
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
                (input: string, ephemeral?: boolean) => {
                    setIsRefreshing(true);
                    const normalizedInput = input.trim();
                    fetchQuery(environment, query, { first: OUTBOX_INITIAL_ITEM_COUNT, subjectSearch: normalizedInput, ephemeral })
                        .subscribe({
                            complete: () => {
                                setIsRefreshing(false);
                                setSearch(input);
                                refetch(
                                    { first: OUTBOX_INITIAL_ITEM_COUNT, subjectSearch: normalizedInput, ephemeral },
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
        fetch(event.target.value, ephemeralFilter);
    };

    const onSearchKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
        if (event.key === 'Enter') {
            fetch.flush();
        }
    };

    const onToggleHideEphemeral = (event: React.ChangeEvent<HTMLInputElement>) => {
        const next = event.target.checked;
        setHideEphemeral(next);
        // Apply immediately rather than waiting out the search throttle.
        fetch(search, next ? false : undefined);
        fetch.flush();
    };

    const outboxes = (data.emailOutboxItems?.edges ?? [])
        .map(edge => edge?.node)
        .filter((node): node is NonNullable<typeof node> => node != null);
    const totalCount = data.emailOutboxItems?.totalCount ?? 0;

    return (
        <Box>
            <AdminAreaBreadcrumbs childRoutes={BREADCRUMB_ROUTES} />
            <Typography variant="h5" gutterBottom>Email Outbox</Typography>
            <Typography variant="body2" sx={{ mb: 2 }}>{DESCRIPTION}</Typography>

            <Box marginBottom={2}>
                <SearchInput
                    fullWidth
                    placeholder="Search by subject"
                    onChange={onSearchChange}
                    onKeyDown={onSearchKeyDown}
                />
            </Box>

            <Paper variant="outlined" sx={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0 }}>
                <Box padding={2} display="flex" alignItems="center" justifyContent="space-between">
                    <Typography variant="subtitle1">
                        {totalCount} {totalCount === 1 ? 'email' : 'emails'}
                    </Typography>
                    <IconButton
                        aria-label="filter emails"
                        aria-haspopup="menu"
                        onClick={(event) => setMenuAnchorEl(event.currentTarget)}
                        sx={{
                            border: `1px solid ${theme.palette.divider}`,
                            borderRadius: 1,
                            height: 40,
                            width: 40,
                        }}
                    >
                        <FilterListIcon color={hideEphemeral ? 'primary' : 'action'} />
                    </IconButton>
                    <Menu
                        anchorEl={menuAnchorEl}
                        open={Boolean(menuAnchorEl)}
                        onClose={() => setMenuAnchorEl(null)}
                        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
                        transformOrigin={{ vertical: 'top', horizontal: 'right' }}
                    >
                        <FormControlLabel
                            sx={{ pl: 2, pr: 2, m: 0 }}
                            control={<Checkbox checked={hideEphemeral} onChange={onToggleHideEphemeral} size="small" />}
                            label="Hide ephemeral emails"
                        />
                    </Menu>
                </Box>
            </Paper>

            <Box sx={{
                borderBottom: `1px solid ${theme.palette.divider}`,
                borderLeft: `1px solid ${theme.palette.divider}`,
                borderRight: `1px solid ${theme.palette.divider}`,
                borderBottomLeftRadius: 4,
                borderBottomRightRadius: 4,
            }}>
                {outboxes.length === 0 ? (
                    <Typography align="center" color="textSecondary" padding={4}>
                        {search ? <>No emails matching <strong>{search}</strong></> : 'No emails to show'}
                    </Typography>
                ) : (
                    <InfiniteScroll
                        dataLength={outboxes.length}
                        next={() => loadNext(20)}
                        hasMore={hasNext}
                        loader={<ListSkeleton rowCount={3} />}
                    >
                        <Box sx={isRefreshing ? { opacity: 0.5 } : undefined}>
                            <ResponsiveTable ariaLabel="emails" columns={COLUMNS}>
                                {outboxes.map(outbox => (
                                    <AdminAreaEmailOutboxListItem key={outbox.id} fragmentRef={outbox} />
                                ))}
                            </ResponsiveTable>
                        </Box>
                    </InfiniteScroll>
                )}
            </Box>
        </Box>
    );
}

function AdminAreaEmailOutboxList() {
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
            <AdminAreaEmailOutboxListContent />
        </Suspense>
    );
}

export default AdminAreaEmailOutboxList;
