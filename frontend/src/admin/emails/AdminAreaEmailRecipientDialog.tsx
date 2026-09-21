import { Box, Button, Dialog, DialogActions, DialogContent, DialogTitle, Divider, SxProps, Theme, Typography, useMediaQuery, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import CopyButton from '../../common/CopyButton';
import Timestamp from '../../common/Timestamp';
import AdminAreaEmailDeliveryStatusChip from './AdminAreaEmailDeliveryStatusChip';
import { AdminAreaEmailRecipientDialogFragment_recipient$key } from './__generated__/AdminAreaEmailRecipientDialogFragment_recipient.graphql';

interface Props {
    fragmentRef: AdminAreaEmailRecipientDialogFragment_recipient$key;
    onClose: () => void;
}

// Field renders a labeled value; renders an em dash when empty.
function Field({ label, action, children, sx }: { label: string; action?: React.ReactNode; children?: React.ReactNode; sx?: SxProps<Theme> }) {
    return (
        <Box sx={sx}>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 1, minHeight: 24 }}>
                <Typography variant="caption" color="textSecondary">{label}</Typography>
                {action}
            </Box>
            {children ?? <Typography variant="body2" color="textSecondary">—</Typography>}
        </Box>
    );
}

function AdminAreaEmailRecipientDialog({ fragmentRef, onClose }: Props) {
    const theme = useTheme();
    const fullScreen = useMediaQuery(theme.breakpoints.down('md'));

    const recipient = useFragment(
        graphql`
            fragment AdminAreaEmailRecipientDialogFragment_recipient on EmailRecipient {
                id
                address
                deliveryStatus
                attemptCount
                availableAt
                lastAttemptAt
                openedAt
                clickedAt
                complainedAt
                failureReason
            }
        `,
        fragmentRef
    );

    // availableAt is a claim lease, not a history field, so it's only meaningful as a future attempt
    // for a recipient that can still be reclaimed; a final status never will be, regardless of its value.
    const isFinal = ['COMPLETED', 'FAILED', 'HARD_BOUNCED', 'ABANDONED'].includes(recipient.deliveryStatus);
    const nextAttempt = !isFinal && new Date(recipient.availableAt) > new Date() ? recipient.availableAt : null;

    return (
        <Dialog open maxWidth="sm" fullWidth fullScreen={fullScreen}>
            <DialogTitle sx={{ wordBreak: 'break-all' }}>{recipient.address}</DialogTitle>
            <DialogContent dividers>
                <Box sx={{
                    display: 'grid',
                    gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' },
                    gap: 2,
                }}>
                    <Field
                        label="ID"
                        action={<CopyButton data={recipient.id} toolTip="Copy recipient ID" />}
                        sx={{ gridColumn: '1 / -1' }}
                    >
                        <Typography variant="body2" sx={{ fontFamily: 'monospace', fontSize: 13, wordBreak: 'break-all' }}>
                            {recipient.id}
                        </Typography>
                    </Field>
                    <Field label="Delivery Status">
                        <AdminAreaEmailDeliveryStatusChip status={recipient.deliveryStatus} />
                    </Field>
                    <Field label="Delivery Attempts">
                        <Typography variant="body2">{recipient.attemptCount}</Typography>
                    </Field>
                    <Field label="Last Attempt">
                        {recipient.lastAttemptAt && <Timestamp timestamp={recipient.lastAttemptAt} format="absolute" variant="body2" />}
                    </Field>
                    {nextAttempt && (
                        <Field label="Next Attempt">
                            <Timestamp timestamp={nextAttempt} format="relative" variant="body2" />
                        </Field>
                    )}
                    <Field label="Opened">
                        {recipient.openedAt && <Timestamp timestamp={recipient.openedAt} format="absolute" variant="body2" />}
                    </Field>
                    <Field label="Clicked">
                        {recipient.clickedAt && <Timestamp timestamp={recipient.clickedAt} format="absolute" variant="body2" />}
                    </Field>
                    <Field label="Complained">
                        {recipient.complainedAt && <Timestamp timestamp={recipient.complainedAt} format="absolute" variant="body2" />}
                    </Field>
                    <Divider sx={{ gridColumn: '1 / -1' }} />
                    <Field
                        label="Failure Reason"
                        action={recipient.failureReason && (
                            <CopyButton data={recipient.failureReason} toolTip="Copy failure reason" />
                        )}
                        sx={{ gridColumn: '1 / -1' }}
                    >
                        {recipient.failureReason && (
                            <Box
                                component="pre"
                                sx={{
                                    m: 0,
                                    p: 1.5,
                                    maxHeight: 240,
                                    overflowY: 'auto',
                                    whiteSpace: 'pre-wrap',
                                    overflowWrap: 'anywhere',
                                    fontFamily: 'monospace',
                                    fontSize: 13,
                                    color: 'error.main',
                                    border: 1,
                                    borderColor: 'divider',
                                    borderRadius: 1,
                                    bgcolor: 'background.default',
                                }}
                            >
                                {recipient.failureReason}
                            </Box>
                        )}
                    </Field>
                </Box>
            </DialogContent>
            <DialogActions>
                <Button color="inherit" onClick={onClose}>Close</Button>
            </DialogActions>
        </Dialog>
    );
}

export default AdminAreaEmailRecipientDialog;
