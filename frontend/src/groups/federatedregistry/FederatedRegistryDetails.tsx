import React, { useState } from 'react';
import { Box, Button, ButtonGroup, Collapse, Dialog, DialogActions, DialogContent, DialogTitle, Divider, Link, Menu, MenuItem, Paper, Stack, styled, Typography } from '@mui/material';
import { ArrowDropUp, ArrowDropDown } from '@mui/icons-material';
import { FederatedRegistryIcon } from '../../common/Icons';
import { LoadingButton } from '@mui/lab';
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import { useFragment, useLazyLoadQuery, useMutation } from 'react-relay/hooks';
import { useNavigate, useParams } from 'react-router-dom';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import TRNButton from '../../common/TRNButton';
import Timestamp from '../../common/Timestamp';
import { GetConnections } from './FederatedRegistryList';
import { FederatedRegistryDetailsDeleteMutation } from './__generated__/FederatedRegistryDetailsDeleteMutation.graphql';
import { FederatedRegistryDetailsFragment_group$key } from './__generated__/FederatedRegistryDetailsFragment_group.graphql';
import { FederatedRegistryDetailsQuery } from './__generated__/FederatedRegistryDetailsQuery.graphql';

const CARD_PADDING = 3;

const FieldLabel = styled(
    Typography
)(() => ({}));

const FieldValue = styled(
    Typography
)(({ theme }) => ({
    color: theme.palette.text.secondary,
    marginBottom: '16px',
    '&:last-child': {
        marginBottom: 0
    }
}));

interface ConfirmationDialogProps {
    hostname: string;
    deleteInProgress: boolean;
    open: boolean;
    onClose: (confirm?: boolean) => void;
}

function DeleteConfirmationDialog(props: ConfirmationDialogProps) {
    const { hostname, deleteInProgress, onClose, open, ...other } = props;
    return (
        <Dialog
            keepMounted={false}
            maxWidth="xs"
            open={open}
            {...other}
        >
            <DialogTitle>Delete Federated Registry</DialogTitle>
            <DialogContent dividers>
                Are you sure you want to delete federated registry <strong>{hostname}</strong>?
            </DialogContent>
            <DialogActions>
                <Button color="inherit" onClick={() => onClose()}>
                    Cancel
                </Button>
                <LoadingButton color="error" loading={deleteInProgress} onClick={() => onClose(true)}>
                    Delete
                </LoadingButton>
            </DialogActions>
        </Dialog>
    );
}

interface Props {
    fragmentRef: FederatedRegistryDetailsFragment_group$key;
}

function FederatedRegistryDetails({ fragmentRef }: Props) {
    const federatedRegistryId = useParams<{ id: string }>().id as string;
    const { enqueueSnackbar } = useSnackbar();
    const navigate = useNavigate();
    const [menuAnchorEl, setMenuAnchorEl] = useState<Element | null>(null);
    const [showDeleteConfirmationDialog, setShowDeleteConfirmationDialog] = useState<boolean>(false);
    const [showMore, setShowMore] = useState(false);

    const group = useFragment<FederatedRegistryDetailsFragment_group$key>(
        graphql`
        fragment FederatedRegistryDetailsFragment_group on Group
        {
            id
            fullPath
        }
        `,
        fragmentRef
    );

    const data = useLazyLoadQuery<FederatedRegistryDetailsQuery>(graphql`
        query FederatedRegistryDetailsQuery($id: String!) {
            node(id: $id) {
                ... on FederatedRegistry {
                    id
                    hostname
                    audience
                    createdBy
                    group {
                        fullPath
                    }
                    metadata {
                        createdAt
                        trn
                    }
                }

            }
        }
    `, { id: federatedRegistryId }, { fetchPolicy: 'store-and-network' });

    const [commit, commitInFlight] = useMutation<FederatedRegistryDetailsDeleteMutation>(graphql`
        mutation FederatedRegistryDetailsDeleteMutation($input: DeleteFederatedRegistryInput!, $connections: [ID!]!) {
            deleteFederatedRegistry(input: $input) {
                federatedRegistry {
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

    const onDeleteConfirmationDialogClosed = (confirm?: boolean) => {
        if (confirm) {
            commit({
                variables: {
                    input: {
                        id: federatedRegistryId
                    },
                    connections: GetConnections(group.id),
                },
                onCompleted: data => {
                    setShowDeleteConfirmationDialog(false);

                    if (data.deleteFederatedRegistry.problems.length) {
                        enqueueSnackbar(data.deleteFederatedRegistry.problems.map((problem: any) => problem.message).join('; '), { variant: 'warning' });
                    } else {
                        navigate(`..`, { replace: true });
                    }
                },
                onError: error => {
                    setShowDeleteConfirmationDialog(false);
                    enqueueSnackbar(`Unexpected error occurred: ${error.message}`, { variant: 'error' });
                }
            });
        } else {
            setShowDeleteConfirmationDialog(false);
        }
    };

    const onOpenMenu = (event: React.MouseEvent<HTMLButtonElement>) => {
        setMenuAnchorEl(event.currentTarget);
    };

    const onMenuClose = () => {
        setMenuAnchorEl(null);
    };

    const onMenuAction = (actionCallback: () => void) => {
        setMenuAnchorEl(null);
        actionCallback();
    };

    if (data.node && federatedRegistryId) {
        const federatedRegistry = data.node;

        return (
            <Box>
                <NamespaceBreadcrumbs
                    namespacePath={group.fullPath}
                    childRoutes={[
                        { title: "federated registries", path: 'federated_registries' },
                        { title: `${federatedRegistryId.substring(0, 8)}...`, path: federatedRegistryId }
                    ]}
                />
                <Paper variant="outlined" sx={{ mt: 3, padding: CARD_PADDING }}>
                    <Box display="flex" justifyContent="space-between">
                        <Box display="flex" alignItems="center">
                            <FederatedRegistryIcon />
                            <Box marginLeft={1}>
                                <Box display="flex" alignItems="center">
                                    <Typography variant="h5" sx={{ mr: 1 }}>{federatedRegistry.hostname}</Typography>
                                </Box>
                            </Box>
                        </Box>
                        <Box>
                            <Stack direction="row" spacing={1}>
                                <TRNButton trn={federatedRegistry.metadata?.trn || ''} />
                                <ButtonGroup variant="outlined" color="primary">
                                    <Button onClick={() => navigate('edit')}>Edit</Button>
                                    <Button
                                        color="primary"
                                        size="small"
                                        aria-label="more options menu"
                                        aria-haspopup="menu"
                                        onClick={onOpenMenu}
                                    >
                                        <ArrowDropDown fontSize="small" />
                                    </Button>
                                </ButtonGroup>
                                <Menu
                                    id="federated-registry-more-options-menu"
                                    anchorEl={menuAnchorEl}
                                    open={Boolean(menuAnchorEl)}
                                    onClose={onMenuClose}
                                    anchorOrigin={{
                                        vertical: 'bottom',
                                        horizontal: 'right',
                                    }}
                                    transformOrigin={{
                                        vertical: 'top',
                                        horizontal: 'right',
                                    }}
                                >
                                    <MenuItem onClick={() => onMenuAction(() => setShowDeleteConfirmationDialog(true))}>
                                        Delete Federated Registry
                                    </MenuItem>
                                </Menu>
                            </Stack>
                        </Box>
                    </Box>
                    <Divider sx={{ opacity: 0.6, my: 2, mx: -CARD_PADDING }} />
                    <Box>
                        <Box sx={{ mt: 2 }}>
                            <FieldLabel>Hostname</FieldLabel>
                            <FieldValue>{federatedRegistry.hostname}</FieldValue>
                            <FieldLabel>Audience</FieldLabel>
                            <FieldValue>{federatedRegistry.audience}</FieldValue>
                        </Box>
                    </Box>
                    <Box sx={{ mt: 2 }}>
                        <Link
                            sx={{ display: 'flex', alignItems: 'center' }}
                            component="button" variant="body1"
                            color="textSecondary"
                            underline="hover"
                            onClick={() => setShowMore(!showMore)}
                        >
                            More Details {showMore ? <ArrowDropUp /> : <ArrowDropDown />}
                        </Link>
                        <Collapse in={showMore} timeout="auto" unmountOnExit>
                            <Box sx={{ mt: 2 }}>
                                <Typography variant="body2">
                                    Created <Timestamp component="span" timestamp={federatedRegistry.metadata?.createdAt} /> by {federatedRegistry.createdBy}
                                </Typography>
                            </Box>
                        </Collapse>
                    </Box>
                </Paper>
                <DeleteConfirmationDialog
                    hostname={federatedRegistry.hostname || ''}
                    deleteInProgress={commitInFlight}
                    open={showDeleteConfirmationDialog}
                    onClose={onDeleteConfirmationDialogClosed}
                />
            </Box>
        );
    }
     else {
        return <Box display="flex" justifyContent="center" sx={{ mt: 4 }}>
            <Typography color="textSecondary">Federated registry with ID {federatedRegistryId} not found</Typography>
        </Box>;
    }
}

export default FederatedRegistryDetails;
