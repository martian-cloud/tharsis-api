import { Box, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { useNavigate } from 'react-router-dom';
import { ResponsiveRow, useCardMode } from '../../common/ResponsiveTable';
import Timestamp from '../../common/Timestamp';
import AdminAreaEmailOutboxLifecycleChip from './AdminAreaEmailOutboxLifecycleChip';
import { AdminAreaEmailOutboxListItemFragment_outbox$key } from './__generated__/AdminAreaEmailOutboxListItemFragment_outbox.graphql';

interface Props {
    fragmentRef: AdminAreaEmailOutboxListItemFragment_outbox$key;
}

function AdminAreaEmailOutboxListItem({ fragmentRef }: Props) {
    const navigate = useNavigate();
    const cardMode = useCardMode();

    const outbox = useFragment(
        graphql`
            fragment AdminAreaEmailOutboxListItemFragment_outbox on EmailOutboxItem {
                id
                subject
                emailType
                ephemeral
                metadata {
                    createdAt
                }
            }
        `,
        fragmentRef
    );

    const onClick = () => navigate(outbox.id);

    const subjectCell = (
        <Typography variant="body1" sx={{ fontWeight: 500, wordBreak: 'break-word' }}>
            {outbox.subject}
        </Typography>
    );

    // ResponsiveRow uses the primary flag to promote the subject to the card header on mobile.
    return (
        <ResponsiveRow
            onClick={onClick}
            cells={[
                { content: subjectCell, primary: true },
                { label: 'Type', content: <Typography variant="body2">{outbox.emailType}</Typography> },
                {
                    label: 'Lifecycle',
                    content: <AdminAreaEmailOutboxLifecycleChip ephemeral={outbox.ephemeral} />,
                },
                {
                    label: 'Created',
                    content: (
                        <Box>
                            <Timestamp timestamp={outbox.metadata.createdAt} format="relative" variant="body2" />
                        </Box>
                    ),
                    align: cardMode ? undefined : 'right',
                },
            ]}
        />
    );
}

export default AdminAreaEmailOutboxListItem;
