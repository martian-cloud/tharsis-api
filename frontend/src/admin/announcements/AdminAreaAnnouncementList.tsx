import { Box, Button, Chip, Dialog, DialogActions, DialogContent, DialogTitle, Paper, Stack, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Typography, useTheme } from '@mui/material';
import { Edit as EditIcon, Delete as DeleteIcon } from '@mui/icons-material';
import { LoadingButton } from '@mui/lab';
import graphql from 'babel-plugin-relay/macro';
import InfiniteScroll from 'react-infinite-scroll-component';
import { useLazyLoadQuery, usePaginationFragment, useMutation } from 'react-relay/hooks';
import { ConnectionHandler } from 'relay-runtime';
import { Link as RouterLink } from 'react-router-dom';
import { useState, useMemo } from 'react';
import { useSnackbar } from 'notistack';
import AdminAreaBreadcrumbs from '../AdminAreaBreadcrumbs';
import Timestamp from '../../common/Timestamp';
import ListSkeleton from '../../skeletons/ListSkeleton';
import AnnouncementAlert from '../../common/AnnouncementAlert';
import { AdminAreaAnnouncementListQuery } from './__generated__/AdminAreaAnnouncementListQuery.graphql';
import { AdminAreaAnnouncementListFragment_announcements$key } from './__generated__/AdminAreaAnnouncementListFragment_announcements.graphql';
import { AnnouncementPaginationQuery } from './__generated__/AnnouncementPaginationQuery.graphql';
import { AdminAreaAnnouncementListDeleteMutation } from './__generated__/AdminAreaAnnouncementListDeleteMutation.graphql';

const DESCRIPTION = 'Announcements allow you to communicate important information to all users across the platform. Create announcements for maintenance windows, feature updates, or other important notices.';
const INITIAL_ITEM_COUNT = 20;

function getConnections(): [string] {
    const connectionId = ConnectionHandler.getConnectionID(
        'root',
        'AdminAreaAnnouncementList_announcements',
        { sort: 'CREATED_AT_DESC' }
    );
    return [connectionId];
}

function getStatusInfo(active: boolean, expired: boolean) {
    if (expired) {
        return { label: 'Expired', color: 'default', variant: 'outlined' };
    } else if (active) {
        return { label: 'Active', color: 'success', variant: 'filled' };
    } else {
        return { label: 'Scheduled', color: 'info', variant: 'outlined' };
    }
}

const query = graphql`
    query AdminAreaAnnouncementListQuery($first: Int!, $after: String) {
        ...AdminAreaAnnouncementListFragment_announcements
    }
`;

interface AnnouncementDeleteConfirmationDialogProps {
    announcement: { id: string; message: string } | null;
    deleteInProgress: boolean;
    onClose: (confirm?: boolean) => void;
}

function AnnouncementDeleteConfirmationDialog(props: AnnouncementDeleteConfirmationDialogProps) {
    const { announcement, deleteInProgress, onClose, ...other } = props;

    return (
        <Dialog
            maxWidth="xs"
            open={!!announcement}
            keepMounted={false}
            {...other}
        >
            <DialogTitle>Delete Announcement</DialogTitle>
            <DialogContent dividers>
                Are you sure you want to delete this announcement?
            </DialogContent>
            <DialogActions>
                <Button color="inherit" onClick={() => onClose()}>
                    Cancel
                </Button>
                <LoadingButton
                    color="error"
                    loading={deleteInProgress}
                    onClick={() => onClose(true)}>
                    Delete
                </LoadingButton>
            </DialogActions>
        </Dialog>
    );
}

function AdminAreaAnnouncementList() {
    const theme = useTheme();
    const { enqueueSnackbar } = useSnackbar();
    const [announcementToDelete, setAnnouncementToDelete] = useState<{ id: string; message: string } | null>(null);

    const queryData = useLazyLoadQuery<AdminAreaAnnouncementListQuery>(
        query,
        { first: INITIAL_ITEM_COUNT },
        { fetchPolicy: 'store-and-network' }
    );

    const { data, loadNext, hasNext } = usePaginationFragment<AnnouncementPaginationQuery, AdminAreaAnnouncementListFragment_announcements$key>(
        graphql`
        fragment AdminAreaAnnouncementListFragment_announcements on Query
        @refetchable(queryName: "AnnouncementPaginationQuery") {
            announcements(
                first: $first
                after: $after
                sort: CREATED_AT_DESC
            ) @connection(key: "AdminAreaAnnouncementList_announcements") {
                totalCount
                edges {
                    node {
                        id
                        message
                        type
                        dismissible
                        startTime
                        endTime
                        active
                        expired
                    }
                }
            }
        }
    `, queryData);

    const [commitDelete, deleteInFlight] = useMutation<AdminAreaAnnouncementListDeleteMutation>(graphql`
        mutation AdminAreaAnnouncementListDeleteMutation($input: DeleteAnnouncementInput!, $connections: [ID!]!) {
            deleteAnnouncement(input: $input) {
                announcement {
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

    const onShowDeleteConfirmationDialog = (announcement: { id: string; message: string }) => {
        setAnnouncementToDelete(announcement);
    };

    const onDelete = (confirm?: boolean) => {
        if (confirm && announcementToDelete) {
            commitDelete({
                variables: {
                    input: {
                        id: announcementToDelete.id
                    },
                    connections: getConnections()
                },
                onCompleted: (data) => {
                    if (data.deleteAnnouncement.problems.length) {
                        enqueueSnackbar(
                            data.deleteAnnouncement.problems.map(p => p.message).join('; '),
                            { variant: 'warning' }
                        );
                    }
                    setAnnouncementToDelete(null);
                },
                onError: (error) => {
                    enqueueSnackbar(`Unexpected error occurred: ${error.message}`, { variant: 'error' });
                    setAnnouncementToDelete(null);
                }
            });
        } else {
            setAnnouncementToDelete(null);
        }
    };

    const announcements = useMemo(() => data.announcements?.edges || [], [data.announcements?.edges]);
    const hasAnnouncements = useMemo(() => announcements.length > 0, [announcements.length]);

    if (!hasAnnouncements) {
        return (
            <Box>
                <AdminAreaBreadcrumbs
                    childRoutes={[
                        { title: "announcements", path: 'announcements' }
                    ]}
                />
                <Box sx={{ marginTop: 4 }} display="flex" justifyContent="center">
                    <Box
                        padding={4}
                        display="flex"
                        flexDirection="column"
                        justifyContent="center"
                        alignItems="center"
                        sx={{ maxWidth: 600 }}
                    >
                        <Typography variant="h6">Get started with announcements</Typography>
                        <Typography color="textSecondary" align="center" sx={{ marginBottom: 2 }}>
                            {DESCRIPTION}
                        </Typography>
                        <Button
                            variant="outlined"
                            component={RouterLink}
                            to="new"
                        >
                            New Announcement
                        </Button>
                    </Box>
                </Box>
            </Box>
        );
    }

    return (
        <Box>
            <AdminAreaBreadcrumbs
                childRoutes={[
                    { title: "announcements", path: 'announcements' }
                ]}
            />
            <Box sx={{
                display: 'flex',
                flexDirection: 'row',
                justifyContent: 'space-between',
                [theme.breakpoints.down('md')]: {
                    flexDirection: 'column',
                    alignItems: 'flex-start',
                    '& > *': { marginBottom: 2 },
                }
            }}>
                <Box>
                    <Typography variant="h5" gutterBottom>Announcements</Typography>
                    <Typography variant="body2">{DESCRIPTION}</Typography>
                </Box>
                <Box>
                    <Button
                        sx={{ minWidth: 200 }}
                        variant="outlined"
                        component={RouterLink}
                        to="new"
                    >
                        New Announcement
                    </Button>
                </Box>
            </Box>

            <Box sx={{ marginTop: 2 }}>
                <Paper sx={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0, border: `1px solid ${theme.palette.divider}` }}>
                    <Box padding={2} display="flex" alignItems="center" justifyContent="space-between">
                        <Typography variant="subtitle1">
                            {data.announcements.totalCount} announcement{data.announcements.totalCount === 1 ? '' : 's'}
                        </Typography>
                    </Box>
                </Paper>
                <InfiniteScroll
                    dataLength={announcements.length ?? 0}
                    next={() => loadNext(20)}
                    hasMore={hasNext}
                    loader={<ListSkeleton rowCount={3} />}
                >
                    <TableContainer>
                        <Table
                            sx={{
                                minWidth: 650,
                                borderCollapse: 'separate',
                                borderSpacing: 0,
                                'td, th': {
                                    borderBottom: `1px solid ${theme.palette.divider}`,
                                },
                                'td:first-of-type, th:first-of-type': {
                                    borderLeft: `1px solid ${theme.palette.divider}`
                                },
                                'td:last-of-type, th:last-of-type': {
                                    borderRight: `1px solid ${theme.palette.divider}`
                                },
                                'tr:last-of-type td:first-of-type': {
                                    borderBottomLeftRadius: 4,
                                },
                                'tr:last-of-type td:last-of-type': {
                                    borderBottomRightRadius: 4
                                }
                            }}
                            aria-label="announcements"
                        >
                            <colgroup>
                                <Box component="col" />
                                <Box component="col" sx={{ width: "120px" }} />
                                <Box component="col" sx={{ width: "150px" }} />
                                <Box component="col" sx={{ width: "150px" }} />
                                <Box component="col" sx={{ width: "120px" }} />
                            </colgroup>
                            <TableHead>
                                <TableRow>
                                    <TableCell>Preview</TableCell>
                                    <TableCell>Status</TableCell>
                                    <TableCell>Start Time</TableCell>
                                    <TableCell>End Time</TableCell>
                                    <TableCell></TableCell>
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {announcements.map((edge: any) => {
                                    const announcement = edge.node;
                                    const statusInfo = getStatusInfo(announcement.active, announcement.expired);

                                    return (
                                        <TableRow key={announcement.id}>
                                            <TableCell>
                                                <AnnouncementAlert
                                                    id={announcement.id}
                                                    message={announcement.message}
                                                    type={announcement.type}
                                                    dismissible={announcement.dismissible}
                                                    onDismiss={() => { /* Preview only - no action */ }}
                                                />
                                            </TableCell>
                                            <TableCell>
                                                <Chip
                                                    label={statusInfo.label}
                                                    color={statusInfo.color as any}
                                                    size="small"
                                                    variant={statusInfo.variant as any}
                                                />
                                            </TableCell>
                                            <TableCell>
                                                <Timestamp
                                                    timestamp={announcement.startTime}
                                                    format="absolute"
                                                    variant="body2"
                                                    color="textSecondary"
                                                />
                                            </TableCell>
                                            <TableCell>
                                                {announcement.endTime ? (
                                                    <Timestamp
                                                        timestamp={announcement.endTime}
                                                        format="absolute"
                                                        variant="body2"
                                                        color="textSecondary"
                                                    />
                                                ) : (
                                                    <Typography variant="body2" color="textSecondary">
                                                        --
                                                    </Typography>
                                                )}
                                            </TableCell>
                                            <TableCell>
                                                <Stack direction="row" spacing={1} display="flex" justifyContent="center">
                                                    <Button
                                                        component={RouterLink}
                                                        to={`${announcement.id}/edit`}
                                                        sx={{ minWidth: 40, padding: '2px' }}
                                                        size="small"
                                                        color="info"
                                                        variant="outlined"
                                                        disabled={deleteInFlight}
                                                    >
                                                        <EditIcon />
                                                    </Button>
                                                    <Button
                                                        onClick={() => onShowDeleteConfirmationDialog({
                                                            id: announcement.id,
                                                            message: announcement.message
                                                        })}
                                                        sx={{ minWidth: 40, padding: '2px' }}
                                                        size="small"
                                                        color="info"
                                                        variant="outlined"
                                                        disabled={deleteInFlight}
                                                    >
                                                        <DeleteIcon />
                                                    </Button>
                                                </Stack>
                                            </TableCell>
                                        </TableRow>
                                    );
                                })}
                            </TableBody>
                        </Table>
                    </TableContainer>
                </InfiniteScroll>
            </Box>

            <AnnouncementDeleteConfirmationDialog
                announcement={announcementToDelete}
                deleteInProgress={deleteInFlight}
                onClose={onDelete}
            />
        </Box>
    );
}

export default AdminAreaAnnouncementList;
