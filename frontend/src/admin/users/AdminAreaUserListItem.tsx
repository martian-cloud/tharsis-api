import { Box, TableCell, TableRow, Typography, Chip, Menu, MenuItem, IconButton } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useFragment, useMutation } from 'react-relay/hooks';
import Gravatar from '../../common/Gravatar';
import Timestamp from '../../common/Timestamp';
import TRNButton from '../../common/TRNButton';
import { AdminAreaUserListItemFragment_user$key } from './__generated__/AdminAreaUserListItemFragment_user.graphql';
import { useContext, useMemo, useState } from 'react';
import { AdminAreaUserListItemUpdateUserAdminStatusMutation } from './__generated__/AdminAreaUserListItemUpdateUserAdminStatusMutation.graphql';
import { useSnackbar } from 'notistack';
import React from 'react';
import ConfirmationDialog from '../../common/ConfirmationDialog';
import { UserContext } from '../../UserContext';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import { Dropdown } from '@mui/base/Dropdown';

interface Props {
    fragmentRef: AdminAreaUserListItemFragment_user$key
}

function AdminAreaUserListItem({ fragmentRef }: Props) {
    const { enqueueSnackbar } = useSnackbar();
    const user = useContext(UserContext);
    const [menuAnchorEl, setMenuAnchorEl] = useState<null | HTMLElement>(null);
    const [showUpdateUserAdminStatusConfirmation, setShowUpdateUserAdminStatusConfirmation] = useState(false);

    const data = useFragment<AdminAreaUserListItemFragment_user$key>(graphql`
        fragment AdminAreaUserListItemFragment_user on User {
            metadata {
                createdAt
                trn
            }
            id
            username
            email
            admin
            active
            scimExternalId
        }
    `, fragmentRef);

    const [commitUpdateUserAdminStatus, commitUpdateUserAdminStatusInFlight] = useMutation<AdminAreaUserListItemUpdateUserAdminStatusMutation>(graphql`
        mutation AdminAreaUserListItemUpdateUserAdminStatusMutation($input: UpdateUserAdminStatusInput!) {
            updateUserAdminStatus(input: $input) {
                user {
                    ...AdminAreaUserListItemFragment_user
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const handleMutationError = (error: Error, completedCallback?: () => void) => {
        if (completedCallback) {
            completedCallback();
        }
        enqueueSnackbar(`Unexpected error: ${error.message}`, { variant: 'error' });
    }

    const handleMutationProblems = (problems: any, completedCallback?: () => void) => {
        if (completedCallback) {
            completedCallback();
        }
        if (problems && problems.length > 0) {
            enqueueSnackbar(problems.map((problem: any) => problem.message).join('; '), { variant: 'warning' });
        }
    }

    const onUpdateUserAdminStatus = () => {
        commitUpdateUserAdminStatus({
            variables: {
                input: {
                    userId: data.id,
                    admin: !data.admin
                }
            },
            onCompleted: data => { handleMutationProblems(data.updateUserAdminStatus?.problems, () => setShowUpdateUserAdminStatusConfirmation(false)) },
            onError: error => { handleMutationError(error, () => setShowUpdateUserAdminStatusConfirmation(false)) }
        })
    }

    function onMenuOpen(event: React.MouseEvent<HTMLButtonElement>) {
        setMenuAnchorEl(event.currentTarget);
    }

    function onMenuClose() {
        setMenuAnchorEl(null);
    }

    const onMenuAction = (actionCallback: () => void) => {
        setMenuAnchorEl(null);
        actionCallback();
    };

    const showDropdownMenuButton = useMemo(() => { return user.email !== data.email && (data.admin || data.active); }, [data])

    return (
        <React.Fragment>
            <TableRow sx={{ '&:last-child td, &:last-child th': { border: 0 } }}>
                <TableCell>
                    <Box display="flex" alignItems="center">
                        <Gravatar width={24} height={24} email={data.email} />
                        <Box ml={2}>
                            <Box display="flex" alignItems="center">
                                <Typography fontWeight={500}>{data.username}</Typography>
                                {data.admin && <Box><Chip sx={{ ml: 1 }} color="secondary" size="xs" label="Admin" /></Box>}
                                {!data.active && <Box><Chip sx={{ ml: 1 }} color="warning" size="xs" label="Inactive" /></Box>}
                            </Box>
                            <Typography color="textSecondary" variant="body2">{data.email}</Typography>
                        </Box>
                    </Box>
                </TableCell>
                <TableCell>
                    {data.scimExternalId ? 'Yes' : 'No'}
                </TableCell>
                <TableCell>
                    <Timestamp variant="body2" timestamp={data.metadata.createdAt} />
                </TableCell>
                <TableCell align="right">
                    <Box display="flex" alignItems="center" justifyContent="flex-end" gap={1}>
                        <TRNButton trn={data.metadata.trn} size="small"/>
                        {showDropdownMenuButton && <Dropdown>
                            <IconButton
                                color="inherit"
                                size="small"
                                onClick={onMenuOpen}
                            >
                                <MoreVertIcon />
                            </IconButton>
                            <Menu
                                id="admin-users-list-more-options-menu"
                                anchorEl={menuAnchorEl}
                                open={Boolean(menuAnchorEl)}
                                onClose={onMenuClose}
                            >
                                <MenuItem
                                    onClick={() => onMenuAction(() => {
                                        setShowUpdateUserAdminStatusConfirmation(true);
                                    })}>
                                    {data.admin ? 'Revoke' : 'Grant'} Admin Permissions
                                </MenuItem>
                            </Menu>
                        </Dropdown>}
                        {!showDropdownMenuButton && <IconButton size="small" sx={{ visibility: 'hidden' }}><MoreVertIcon /></IconButton>}
                    </Box>
                </TableCell>
            </TableRow>
            {showUpdateUserAdminStatusConfirmation && <ConfirmationDialog
                title={data.admin ? 'Revoke Admin Permissions' : 'Grant Admin Permissions'}
                message={<React.Fragment>Are you sure you want to {data.admin ? 'revoke' : 'grant'} admin permissions {data.admin ? 'from' : 'to'} user <strong>{data.username}</strong>?</React.Fragment>}
                confirmButtonLabel={data.admin ? 'Revoke' : 'Grant'}
                opInProgress={commitUpdateUserAdminStatusInFlight}
                onConfirm={onUpdateUserAdminStatus}
                onClose={() => setShowUpdateUserAdminStatusConfirmation(false)}
            />}
        </React.Fragment>
    );
}

export default AdminAreaUserListItem;
