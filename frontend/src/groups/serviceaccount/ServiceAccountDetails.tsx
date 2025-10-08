import { ArrowDropUp } from '@mui/icons-material';
import { default as ArrowDropDown, default as ArrowDropDownIcon } from '@mui/icons-material/ArrowDropDown';
import { LoadingButton } from '@mui/lab';
import { Avatar, ButtonGroup, Chip, Collapse, Dialog, DialogActions, DialogContent, DialogTitle, Link, Menu, MenuItem, Paper, Stack, Table, TableBody, TableCell, TableHead, TableRow } from '@mui/material';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import teal from '@mui/material/colors/teal';
import Divider from '@mui/material/Divider';
import Typography from '@mui/material/Typography';
import graphql from 'babel-plugin-relay/macro';
import moment from 'moment';
import { useSnackbar } from 'notistack';
import React, { useState } from 'react';
import { useFragment, useLazyLoadQuery, useMutation } from "react-relay/hooks";
import { useNavigate, useParams } from 'react-router-dom';
import TRNButton from '../../common/TRNButton';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import { GetConnections } from './ServiceAccountList';
import { ServiceAccountDetailsDeleteMutation } from './__generated__/ServiceAccountDetailsDeleteMutation.graphql';
import { ServiceAccountDetailsFragment_group$key } from './__generated__/ServiceAccountDetailsFragment_group.graphql';
import { ServiceAccountDetailsQuery } from './__generated__/ServiceAccountDetailsQuery.graphql';

const CARD_PADDING = 3;

interface Props {
    fragmentRef: ServiceAccountDetailsFragment_group$key
}

interface ConfirmationDialogProps {
    serviceAccountPath: string
    deleteInProgress: boolean;
    keepMounted: boolean;
    open: boolean;
    onClose: (confirm?: boolean) => void
}

function DeleteConfirmationDialog(props: ConfirmationDialogProps) {
    const { serviceAccountPath, deleteInProgress, onClose, open, ...other } = props;
    return (
        <Dialog
            maxWidth="xs"
            open={open}
            {...other}
        >
            <DialogTitle>Delete Service Account</DialogTitle>
            <DialogContent dividers>
                Are you sure you want to delete service account <strong>{serviceAccountPath}</strong>?
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

function ServiceAccountDetails(props: Props) {
    const { id } = useParams();
    const { enqueueSnackbar } = useSnackbar();
    const navigate = useNavigate();
    const [menuAnchorEl, setMenuAnchorEl] = useState<Element | null>(null);
    const [showDeleteConfirmationDialog, setShowDeleteConfirmationDialog] = useState<boolean>(false);
    const [showMore, setShowMore] = useState(false);

    const serviceAccountId = id as string;

    const group = useFragment<ServiceAccountDetailsFragment_group$key>(
        graphql`
        fragment ServiceAccountDetailsFragment_group on Group
        {
          id
          fullPath
        }
        `, props.fragmentRef);

    const data = useLazyLoadQuery<ServiceAccountDetailsQuery>(graphql`
        query ServiceAccountDetailsQuery($id: String!) {
            serviceAccount(id: $id) {
                metadata {
                    createdAt
                    trn
                }
                id
                name
                description
                resourcePath
                createdBy
                oidcTrustPolicies {
                    issuer
                    boundClaimsType
                    boundClaims {
                        name
                        value
                    }
                }
            }
        }
    `, { id: serviceAccountId }, { fetchPolicy: 'store-and-network' });

    const [commit, commitInFlight] = useMutation<ServiceAccountDetailsDeleteMutation>(graphql`
        mutation ServiceAccountDetailsDeleteMutation($input: DeleteServiceAccountInput!, $connections: [ID!]!) {
            deleteServiceAccount(input: $input) {
                serviceAccount {
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
                        id: serviceAccountId
                    },
                    connections: GetConnections(group.id),
                },
                onCompleted: data => {
                    setShowDeleteConfirmationDialog(false);

                    if (data.deleteServiceAccount.problems.length) {
                        enqueueSnackbar(data.deleteServiceAccount.problems.map((problem: any) => problem.message).join('; '), { variant: 'warning' });
                    } else {
                        navigate(`..`);
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

    if (data.serviceAccount && id) {
        return (
            <Box>
                <NamespaceBreadcrumbs
                    namespacePath={group.fullPath}
                    childRoutes={[
                        { title: "service accounts", path: 'service_accounts' },
                        { title: data.serviceAccount.name, path: id }
                    ]}
                />
                <Paper variant="outlined" sx={{ marginTop: 3, padding: CARD_PADDING }}>
                    <Box display="flex" justifyContent="space-between">
                        <Box display="flex" alignItems="center">
                            <Avatar variant="rounded" sx={{ width: 32, height: 32, marginRight: 1, bgcolor: teal[200] }}>
                                {data.serviceAccount.name[0].toUpperCase()}
                            </Avatar>
                            <Box marginLeft={1}>
                                <Box display="flex" alignItems="center">
                                    <Typography variant="h5" sx={{ marginRight: 1 }}>{data.serviceAccount.name}</Typography>
                                </Box>
                                <Typography color="textSecondary">{data.serviceAccount.description}</Typography>
                            </Box>
                        </Box>
                        <Box>
                            <Stack direction="row" spacing={1}>
                                <TRNButton trn={data.serviceAccount.metadata.trn} />
                                <ButtonGroup variant="outlined" color="primary">
                                    <Button onClick={() => navigate('edit')}>Edit</Button>
                                    <Button
                                        color="primary"
                                        size="small"
                                        aria-label="more options menu"
                                        aria-haspopup="menu"
                                        onClick={onOpenMenu}
                                    >
                                        <ArrowDropDownIcon fontSize="small" />
                                    </Button>
                                </ButtonGroup>
                                <Menu
                                    id="service-account-more-options-menu"
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
                                        Delete Service Account
                                    </MenuItem>
                                </Menu>
                            </Stack>
                        </Box>
                    </Box>
                    <Divider sx={{ opacity: 0.6, marginTop: 2, marginBottom: 2, marginLeft: -CARD_PADDING, marginRight: -CARD_PADDING }} />
                    <Box>
                        <Typography>Trusted Identity Providers</Typography>
                        <Typography variant="caption">
                            Tokens issued by the following identity providers will be able to login to this service account provided that the bound claims match the token claims
                        </Typography>
                        <Paper sx={{ marginTop: 2, padding: 1 }}>
                            <Table>
                                <TableHead>
                                    <TableRow>
                                        <TableCell>Issuer URL</TableCell>
                                        <TableCell>Bound Claims</TableCell>
                                        <TableCell>Wildcard Match Enabled</TableCell>
                                    </TableRow>
                                </TableHead>
                                <TableBody>
                                    {data.serviceAccount.oidcTrustPolicies.map((trustPolicy, index) => (<TableRow
                                        key={index}
                                        sx={{ '&:last-child td, &:last-child th': { border: 0 }, height: 64 }}>
                                        <TableCell>{trustPolicy.issuer}</TableCell>
                                        <TableCell>
                                            <Box
                                                display="flex"
                                                flexWrap="wrap"
                                                sx={{
                                                    margin: '0 -4px',
                                                    '& > *': {
                                                        margin: '4px'
                                                    },
                                                }}
                                            >
                                                {trustPolicy.boundClaims.map(claim => (
                                                    <Chip
                                                        size="small"
                                                        key={claim.name}
                                                        variant="outlined"
                                                        label={<React.Fragment>
                                                            <Typography variant="body2" component="span" sx={{ fontWeight: 'bold' }}>{claim.name}:</Typography>
                                                            <Typography variant="body2" component="span">{' ' + claim.value}</Typography>
                                                        </React.Fragment>}
                                                    />
                                                ))}
                                            </Box>
                                        </TableCell>
                                        <TableCell>{trustPolicy.boundClaimsType === 'GLOB' ? 'Yes' : 'No'}</TableCell>
                                    </TableRow>))}
                                </TableBody>
                            </Table>
                        </Paper>
                        <Box marginTop={4}>
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
                                <Box marginTop={2}>
                                    <Typography variant="body2">
                                        Created {moment(data.serviceAccount.metadata.createdAt as moment.MomentInput).fromNow()} by {data.serviceAccount.createdBy}
                                    </Typography>
                                </Box>
                            </Collapse>
                        </Box>
                    </Box>
                </Paper >
                <DeleteConfirmationDialog
                    serviceAccountPath={data.serviceAccount.resourcePath}
                    keepMounted
                    deleteInProgress={commitInFlight}
                    open={showDeleteConfirmationDialog}
                    onClose={onDeleteConfirmationDialogClosed}
                />
            </Box >
        );
    } else {
        return <Box display="flex" justifyContent="center" marginTop={4}>
            <Typography color="textSecondary">Service account with ID {serviceAccountId} not found</Typography>
        </Box>;
    }
}

export default ServiceAccountDetails
