import { Box, Chip, Paper, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import throttle from 'lodash.throttle';
import { useEffect, useMemo, useState } from 'react';
import InfiniteScroll from 'react-infinite-scroll-component';
import { fetchQuery, usePaginationFragment, useRelayEnvironment } from 'react-relay/hooks';
import SearchInput from '../../common/SearchInput';
import { ResponsiveTable } from '../../common/ResponsiveTable';
import ListSkeleton from '../../skeletons/ListSkeleton';
import AdminAreaEmailRecipientListItem from './AdminAreaEmailRecipientListItem';
import paginationQueryNode from './__generated__/AdminAreaEmailRecipientListPaginationQuery.graphql';
import { AdminAreaEmailRecipientListFragment_outbox$key } from './__generated__/AdminAreaEmailRecipientListFragment_outbox.graphql';
import { AdminAreaEmailRecipientListPaginationQuery, EmailDeliveryStatus } from './__generated__/AdminAreaEmailRecipientListPaginationQuery.graphql';

const RECIPIENTS_INITIAL_ITEM_COUNT = 20;

// RecipientFilter carries every filter arg (null when inactive) so each refetch fully overrides the previous filter instead of merging stale vars.
interface RecipientFilter {
    deliveryStatuses: EmailDeliveryStatus[] | null;
    hasOpened: boolean | null;
    hasIssues: boolean | null;
}

const COLUMNS = [
    { label: 'Address' },
    { label: 'Delivery status' },
    { label: 'Opened' },
    { label: 'Clicked' },
    { label: 'Complained' },
];

interface Props {
    outboxId: string;
    fragmentRef: AdminAreaEmailRecipientListFragment_outbox$key;
    recipientCount: number;
    // ephemeral gates the retention footer note: day-based cleanup only applies to ephemeral emails.
    ephemeral: boolean;
    // retentionDays is how long an ephemeral email's delivery events are kept before cleanup.
    retentionDays: number;
}

const FILTER_PILLS: { key: string; label: string; filter: RecipientFilter }[] = [
    { key: 'all', label: 'All', filter: { deliveryStatuses: null, hasOpened: null, hasIssues: null } },
    { key: 'delivered', label: 'Delivered', filter: { deliveryStatuses: ['COMPLETED'], hasOpened: null, hasIssues: null } },
    { key: 'opened', label: 'Opened', filter: { deliveryStatuses: null, hasOpened: true, hasIssues: null } },
    { key: 'issues', label: 'Issues', filter: { deliveryStatuses: null, hasOpened: null, hasIssues: true } },
];

function AdminAreaEmailRecipientList({ outboxId, fragmentRef, recipientCount, ephemeral, retentionDays }: Props) {
    const environment = useRelayEnvironment();
    const [search, setSearch] = useState('');
    const [activePill, setActivePill] = useState('all');
    const [isRefreshing, setIsRefreshing] = useState(false);

    const { data, loadNext, hasNext, refetch } = usePaginationFragment<
        AdminAreaEmailRecipientListPaginationQuery,
        AdminAreaEmailRecipientListFragment_outbox$key
    >(
        graphql`
            fragment AdminAreaEmailRecipientListFragment_outbox on EmailOutboxItem
            @refetchable(queryName: "AdminAreaEmailRecipientListPaginationQuery") {
                recipients(first: $first, after: $after, search: $search, deliveryStatuses: $deliveryStatuses, hasOpened: $hasOpened, hasIssues: $hasIssues, sort: CREATED_AT_ASC)
                @connection(key: "AdminAreaEmailRecipientList_recipients") {
                    edges {
                        node {
                            id
                            ...AdminAreaEmailRecipientListItemFragment_recipient
                        }
                    }
                }
            }
        `,
        fragmentRef
    );

    const fetch = useMemo(
        () =>
            throttle(
                (input: string, filter: RecipientFilter) => {
                    setIsRefreshing(true);
                    const normalizedInput = input.trim();
                    // Prefetch into the store first so the 'store-only' refetch below never suspends
                    // (a suspending refetch here would blow away the whole outbox details page).
                    fetchQuery<AdminAreaEmailRecipientListPaginationQuery>(environment, paginationQueryNode, { id: outboxId, first: RECIPIENTS_INITIAL_ITEM_COUNT, search: normalizedInput, ...filter })
                        .subscribe({
                            complete: () => {
                                setIsRefreshing(false);
                                setSearch(input);
                                refetch(
                                    { first: RECIPIENTS_INITIAL_ITEM_COUNT, search: normalizedInput, ...filter },
                                    { fetchPolicy: 'store-only' }
                                );
                            },
                            error: () => setIsRefreshing(false),
                        });
                },
                2000,
                { leading: false, trailing: true }
            ),
        [environment, refetch, outboxId]
    );

    useEffect(() => () => { fetch.cancel(); }, [fetch]);

    const activeFilter = useMemo(
        () => FILTER_PILLS.find(pill => pill.key === activePill)?.filter
            ?? { deliveryStatuses: null, hasOpened: null, hasIssues: null },
        [activePill]
    );

    const onSearchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        fetch(event.target.value, activeFilter);
    };

    const onSearchKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
        if (event.key === 'Enter') {
            fetch.flush();
        }
    };

    const onFilterPillClick = (key: string, filter: RecipientFilter) => {
        setActivePill(key);
        fetch(search, filter);
        fetch.flush();
    };

    const recipients = (data.recipients.edges ?? [])
        .map(edge => edge?.node)
        .filter((node): node is NonNullable<typeof node> => node != null);
    // Use the stats aggregate for the total rather than the connection's totalCount, which would fire a separate COUNT query per load.
    const totalCount = recipientCount;

    return (
        <Paper sx={{ p: 3 }}>
            <Box sx={{
                display: 'flex',
                flexDirection: { xs: 'column', md: 'row' },
                alignItems: { md: 'center' },
                justifyContent: 'space-between',
                gap: 2,
                mb: 2,
            }}>
                <Typography variant="h6" sx={{ flexShrink: 0 }}>Recipients</Typography>
                <Box sx={{
                    display: 'flex',
                    flexDirection: { xs: 'column', sm: 'row' },
                    alignItems: { sm: 'center' },
                    gap: 1,
                }}>
                    <Box display="flex" gap={1} flexWrap="wrap">
                        {FILTER_PILLS.map(pill => (
                            <Chip
                                key={pill.key}
                                label={pill.label}
                                onClick={() => onFilterPillClick(pill.key, pill.filter)}
                                color={activePill === pill.key ? 'primary' : 'default'}
                                variant={activePill === pill.key ? 'filled' : 'outlined'}
                            />
                        ))}
                    </Box>
                    <SearchInput
                        placeholder="Search by address"
                        onChange={onSearchChange}
                        onKeyDown={onSearchKeyDown}
                    />
                </Box>
            </Box>
            {recipients.length === 0 ? (
                <Paper variant="outlined" sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
                    <Typography color="textSecondary">
                        {search ? <>No recipients matching <strong>{search}</strong></> : 'No recipients'}
                    </Typography>
                </Paper>
            ) : (
                <InfiniteScroll
                    dataLength={recipients.length}
                    next={() => loadNext(20)}
                    hasMore={hasNext}
                    loader={<ListSkeleton rowCount={3} />}
                >
                    <Box sx={isRefreshing ? { opacity: 0.5 } : undefined}>
                        <ResponsiveTable ariaLabel="recipients" columns={COLUMNS}>
                            {recipients.map(recipient => (
                                <AdminAreaEmailRecipientListItem key={recipient.id} fragmentRef={recipient} />
                            ))}
                        </ResponsiveTable>
                    </Box>
                </InfiniteScroll>
            )}
            <Box sx={{
                display: 'flex',
                flexDirection: { xs: 'column', sm: 'row' },
                justifyContent: 'space-between',
                gap: 1,
                mt: 2,
                pt: 2,
                borderTop: 1,
                borderColor: 'divider',
            }}>
                <Typography variant="body2" color="textSecondary">
                    {totalCount} {totalCount === 1 ? 'recipient' : 'recipients'}
                </Typography>
                <Typography variant="body2" color="textSecondary">
                    {ephemeral
                        ? `Delivery events are retained for ${retentionDays} day${retentionDays === 1 ? '' : 's'}.`
                        : 'Delivery events are kept indefinitely.'}
                </Typography>
            </Box>
        </Paper>
    );
}

export { RECIPIENTS_INITIAL_ITEM_COUNT };
export default AdminAreaEmailRecipientList;
