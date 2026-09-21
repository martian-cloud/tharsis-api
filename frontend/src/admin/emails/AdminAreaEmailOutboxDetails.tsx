import EmailOutlinedIcon from '@mui/icons-material/EmailOutlined';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import { Box, CircularProgress, Paper, Stack, Tooltip, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { Suspense } from 'react';
import { useLazyLoadQuery } from 'react-relay/hooks';
import { useParams } from 'react-router-dom';
import Timestamp from '../../common/Timestamp';
import TRNButton from '../../common/TRNButton';
import AdminAreaBreadcrumbs from '../AdminAreaBreadcrumbs';
import AdminAreaEmailOutboxLifecycleChip from './AdminAreaEmailOutboxLifecycleChip';
import AdminAreaEmailRecipientList, { RECIPIENTS_INITIAL_ITEM_COUNT } from './AdminAreaEmailRecipientList';
import { AdminAreaEmailOutboxDetailsQuery } from './__generated__/AdminAreaEmailOutboxDetailsQuery.graphql';

const query = graphql`
    query AdminAreaEmailOutboxDetailsQuery($id: String!, $first: Int, $after: String, $search: String, $deliveryStatuses: [EmailDeliveryStatus!], $hasOpened: Boolean, $hasIssues: Boolean) {
        config {
            emailEphemeralRetentionDays
        }
        node(id: $id) {
            ... on EmailOutboxItem {
                id
                subject
                emailType
                ephemeral
                status
                metadata {
                    createdAt
                    trn
                }
                recipientStats {
                    total
                    delivered
                    opened
                    clicked
                    issues
                }
                ...AdminAreaEmailRecipientListFragment_outbox
            }
        }
    }
`;

const STATUS_META: Record<string, { label: string; color: string }> = {
    PREPARING: { label: 'Preparing recipients', color: 'text.secondary' },
    READY: { label: 'Sending', color: 'info.main' },
    COMPLETED: { label: 'All deliveries complete', color: 'success.main' },
    FAILED: { label: 'Failed to send', color: 'error.main' },
};

// InfoCard renders a labeled value as its own card in the top info row.
function InfoCard({ label, children }: { label: string; children: React.ReactNode }) {
    return (
        <Paper sx={{ p: 2 }}>
            <Typography variant="caption" color="textSecondary">{label}</Typography>
            <Box mt={0.5}>{children}</Box>
        </Paper>
    );
}

// StatCard renders one big-number delivery stat in the stats row.
function StatCard({ label, value, color, description }: { label: string; value: number; color?: string; description?: string }) {
    return (
        <Paper sx={{ p: 2 }}>
            <Box display="flex" alignItems="center" gap={0.5}>
                <Typography variant="caption" color="textSecondary">{label}</Typography>
                {description && (
                    <Tooltip title={description}>
                        <InfoOutlinedIcon sx={{ fontSize: 14, color: 'text.secondary' }} />
                    </Tooltip>
                )}
            </Box>
            <Typography variant="h5" color={color}>{value}</Typography>
        </Paper>
    );
}

function AdminAreaEmailOutboxDetailsContent() {
    const theme = useTheme();
    const outboxId = useParams().outboxId as string;

    const queryData = useLazyLoadQuery<AdminAreaEmailOutboxDetailsQuery>(
        query,
        { id: outboxId, first: RECIPIENTS_INITIAL_ITEM_COUNT },
        { fetchPolicy: 'store-and-network' }
    );

    const outbox = queryData.node;

    const breadcrumbs = [
        { title: 'email outbox', path: 'email_outbox' },
        { title: `${outboxId.substring(0, 8)}...`, path: outboxId },
    ];

    if (!outbox?.id || !outbox.metadata || !outbox.recipientStats) {
        return (
            <Box>
                <AdminAreaBreadcrumbs childRoutes={breadcrumbs} />
                <Box display="flex" justifyContent="center" sx={{ mt: 4 }}>
                    <Typography color="textSecondary">Email not found</Typography>
                </Box>
            </Box>
        );
    }

    const statusMeta = STATUS_META[outbox.status ?? ''];

    const stats = outbox.recipientStats;
    const statCards: { label: string; value: number; color?: string; description?: string }[] = [
        { label: 'Recipients', value: stats.total },
        { label: 'Delivered', value: stats.delivered, color: 'success.main', description: "Recipients the email provider confirmed delivery for, or accepted when the provider doesn't report delivery." },
        { label: 'Opened', value: stats.opened, description: 'Recipients who opened the email. Requires open tracking; stays zero when tracking is unavailable.' },
        { label: 'Clicked', value: stats.clicked, description: 'Recipients who clicked a link in the email. Requires click tracking; stays zero when tracking is unavailable.' },
        { label: 'Issues', value: stats.issues, color: stats.issues > 0 ? 'error.main' : undefined, description: 'Recipients that bounced, were abandoned, failed, or complained.' },
    ];

    return (
        <Box>
            <AdminAreaBreadcrumbs childRoutes={breadcrumbs} />

            <Box sx={{
                display: 'flex',
                flexDirection: 'row',
                justifyContent: 'space-between',
                [theme.breakpoints.down('sm')]: {
                    flexDirection: 'column',
                    alignItems: 'flex-start',
                    '& > *': { mb: 2 },
                },
            }}>
                <Box display="flex" alignItems="center" mb={2} sx={{ minWidth: 0 }}>
                    <EmailOutlinedIcon sx={{ mr: 2 }} />
                    <Box sx={{ minWidth: 0 }}>
                        <Box display="flex" alignItems="center" gap={1}>
                            <Typography variant="h5" sx={{ wordBreak: 'break-word' }}>{outbox.subject}</Typography>
                        </Box>
                        <Box display="flex" alignItems="center" flexWrap="wrap" gap={0.5}>
                            <Typography variant="caption" color="textSecondary">
                                Created <Timestamp timestamp={outbox.metadata.createdAt} />
                            </Typography>
                        </Box>
                    </Box>
                </Box>
                <Box>
                    <Stack direction="row" spacing={1}>
                        <TRNButton trn={outbox.metadata.trn} />
                    </Stack>
                </Box>
            </Box>

            <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: 'repeat(3, 1fr)' }, gap: 2, mb: 3 }}>
                <InfoCard label="Type">
                    <Typography color="textSecondary">{outbox.emailType}</Typography>
                </InfoCard>
                <InfoCard label="Lifecycle">
                    <AdminAreaEmailOutboxLifecycleChip
                        ephemeral={outbox.ephemeral}
                        retentionDays={queryData.config.emailEphemeralRetentionDays}
                    />
                </InfoCard>
                <InfoCard label="Status">
                    {statusMeta ? (
                        <Typography color="textSecondary" display="flex" alignItems="center" gap={0.5}>
                            <Box component="span" sx={{ width: 8, height: 8, borderRadius: '50%', backgroundColor: statusMeta.color, display: 'inline-block' }} />
                            {statusMeta.label}
                        </Typography>
                    ) : (
                        <Typography color="textSecondary">{outbox.status}</Typography>
                    )}
                </InfoCard>
            </Box>

            <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: 'repeat(5, 1fr)' }, gap: 2, mb: 3 }}>
                {statCards.map(card => (
                    <StatCard key={card.label} label={card.label} value={card.value} color={card.color} description={card.description} />
                ))}
            </Box>

            <AdminAreaEmailRecipientList
                outboxId={outbox.id}
                fragmentRef={outbox}
                ephemeral={!!outbox.ephemeral}
                retentionDays={queryData.config.emailEphemeralRetentionDays}
                recipientCount={outbox.recipientStats.total}
            />
        </Box>
    );
}

function AdminAreaEmailOutboxDetails() {
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
            <AdminAreaEmailOutboxDetailsContent />
        </Suspense>
    );
}

export default AdminAreaEmailOutboxDetails;
