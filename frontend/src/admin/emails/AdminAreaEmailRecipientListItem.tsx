import { Box, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useState } from 'react';
import { useFragment } from 'react-relay/hooks';
import Gravatar from '../../common/Gravatar';
import { ResponsiveRow } from '../../common/ResponsiveTable';
import Timestamp from '../../common/Timestamp';
import AdminAreaEmailDeliveryStatusChip from './AdminAreaEmailDeliveryStatusChip';
import AdminAreaEmailRecipientDialog from './AdminAreaEmailRecipientDialog';
import { AdminAreaEmailRecipientListItemFragment_recipient$key } from './__generated__/AdminAreaEmailRecipientListItemFragment_recipient.graphql';

interface Props {
    fragmentRef: AdminAreaEmailRecipientListItemFragment_recipient$key;
}

function AdminAreaEmailRecipientListItem({ fragmentRef }: Props) {
    const [dialogOpen, setDialogOpen] = useState(false);

    const recipient = useFragment(
        graphql`
            fragment AdminAreaEmailRecipientListItemFragment_recipient on EmailRecipient {
                address
                deliveryStatus
                openedAt
                clickedAt
                complainedAt
                ...AdminAreaEmailRecipientDialogFragment_recipient
            }
        `,
        fragmentRef
    );

    return (
        <>
            <ResponsiveRow
                onClick={() => setDialogOpen(true)}
                cells={[
                    {
                        content: (
                            <Box display="flex" alignItems="center">
                                <Gravatar width={24} height={24} email={recipient.address} />
                                <Typography variant="body2" sx={{ ml: 2, wordBreak: 'break-all', fontWeight: 500 }}>
                                    {recipient.address}
                                </Typography>
                            </Box>
                        ),
                        primary: true,
                    },
                    { label: 'Delivery status', content: <AdminAreaEmailDeliveryStatusChip status={recipient.deliveryStatus} /> },
                    {
                        label: 'Opened',
                        content: recipient.openedAt
                            ? <Timestamp timestamp={recipient.openedAt} format="relative" variant="body2" />
                            : <Typography variant="body2" color="textSecondary">—</Typography>,
                    },
                    {
                        label: 'Clicked',
                        content: recipient.clickedAt
                            ? <Timestamp timestamp={recipient.clickedAt} format="relative" variant="body2" />
                            : <Typography variant="body2" color="textSecondary">—</Typography>,
                    },
                    {
                        label: 'Complained',
                        content: recipient.complainedAt
                            ? <Typography variant="body2" color="error.main">Yes</Typography>
                            : <Typography variant="body2" color="textSecondary">—</Typography>,
                    },
                ]}
            />
            {dialogOpen && (
                <AdminAreaEmailRecipientDialog fragmentRef={recipient} onClose={() => setDialogOpen(false)} />
            )}
        </>
    );
}

export default AdminAreaEmailRecipientListItem;
