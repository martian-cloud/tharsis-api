import MoreVertIcon from '@mui/icons-material/MoreVert';
import { Box, IconButton, Menu, MenuItem, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import React, { useState } from 'react';
import { useFragment, useMutation } from 'react-relay/hooks';
import ConfirmationDialog from '../../common/ConfirmationDialog';
import { ResponsiveRow } from '../../common/ResponsiveTable';
import Timestamp from '../../common/Timestamp';
import TRNButton from '../../common/TRNButton';
import AdminAreaEmailSuppressionCauseChip from './AdminAreaEmailSuppressionCauseChip';
import { AdminAreaEmailSuppressionListItemDeleteMutation } from './__generated__/AdminAreaEmailSuppressionListItemDeleteMutation.graphql';
import { AdminAreaEmailSuppressionListItemFragment_suppression$key } from './__generated__/AdminAreaEmailSuppressionListItemFragment_suppression.graphql';

interface Props {
    fragmentRef: AdminAreaEmailSuppressionListItemFragment_suppression$key;
    connectionId: string;
}

function AdminAreaEmailSuppressionListItem({ fragmentRef, connectionId }: Props) {
    const { enqueueSnackbar } = useSnackbar();
    const [menuAnchorEl, setMenuAnchorEl] = useState<null | HTMLElement>(null);
    const [showConfirmation, setShowConfirmation] = useState(false);

    const suppression = useFragment(
        graphql`
            fragment AdminAreaEmailSuppressionListItemFragment_suppression on EmailSuppression {
                id
                address
                cause
                metadata {
                    createdAt
                    trn
                }
            }
        `,
        fragmentRef
    );

    const [commitDelete, deleteInFlight] = useMutation<AdminAreaEmailSuppressionListItemDeleteMutation>(graphql`
        mutation AdminAreaEmailSuppressionListItemDeleteMutation($input: DeleteEmailSuppressionInput!, $connections: [ID!]!) {
            deleteEmailSuppression(input: $input) {
                suppression {
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

    const onRemove = () => {
        commitDelete({
            variables: {
                input: { id: suppression.id },
                connections: [connectionId],
            },
            onCompleted: data => {
                setShowConfirmation(false);
                if (data.deleteEmailSuppression.problems.length) {
                    enqueueSnackbar(data.deleteEmailSuppression.problems.map(p => p.message).join('; '), { variant: 'warning' });
                }
            },
            onError: error => {
                setShowConfirmation(false);
                enqueueSnackbar(`Unexpected error: ${error.message}`, { variant: 'error' });
            },
        });
    };

    const actions = (
        <Box display="flex" alignItems="center" gap={1} justifyContent="flex-end">
            <TRNButton trn={suppression.metadata.trn} size="small" />
            <IconButton
                aria-label="suppression options"
                aria-haspopup="menu"
                size="small"
                onClick={event => setMenuAnchorEl(event.currentTarget)}
            >
                <MoreVertIcon />
            </IconButton>
            <Menu
                anchorEl={menuAnchorEl}
                open={Boolean(menuAnchorEl)}
                onClose={() => setMenuAnchorEl(null)}
            >
                <MenuItem
                    onClick={() => {
                        setMenuAnchorEl(null);
                        setShowConfirmation(true);
                    }}
                >
                    Remove suppression
                </MenuItem>
            </Menu>
        </Box>
    );

    return (
        <React.Fragment>
            <ResponsiveRow
                cells={[
                    {
                        primary: true,
                        content: (
                            <Typography variant="body2" sx={{ wordBreak: 'break-all', fontWeight: 500 }}>
                                {suppression.address}
                            </Typography>
                        ),
                    },
                    { label: 'Cause', content: <AdminAreaEmailSuppressionCauseChip cause={suppression.cause} /> },
                    {
                        label: 'Created',
                        content: <Timestamp timestamp={suppression.metadata.createdAt} format="relative" variant="body2" />,
                    },
                    { align: 'right', content: actions },
                ]}
            />
            {showConfirmation && (
                <ConfirmationDialog
                    title="Remove Suppression"
                    confirmLabel="Remove"
                    confirmInProgress={deleteInFlight}
                    onConfirm={onRemove}
                    onClose={() => setShowConfirmation(false)}
                >
                    Are you sure you want to remove the suppression for <strong>{suppression.address}</strong>? The system will be able to send email to this address again. This does not remove the address from the email provider's own suppression list.
                </ConfirmationDialog>
            )}
        </React.Fragment>
    );
}

export default AdminAreaEmailSuppressionListItem;
