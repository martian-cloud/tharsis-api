import { default as ArrowDropDownIcon } from '@mui/icons-material/ArrowDropDown';
import { Alert, Avatar, ButtonGroup, Chip, Menu, MenuItem, Paper, Stack, Tab, Tabs, Table, TableBody, TableCell, TableHead, TableRow } from '@mui/material';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Typography from '@mui/material/Typography';
import graphql from 'babel-plugin-relay/macro';
import moment, { Moment } from 'moment';
import { useSnackbar } from 'notistack';
import React, { useContext, useState } from 'react';
import { useFragment, useLazyLoadQuery, useMutation } from "react-relay/hooks";
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { ApiConfigContext } from '../../ApiConfigContext';
import ConfirmationDialog from '../../common/ConfirmationDialog';
import CopyButton from '../../common/CopyButton';
import Timestamp from '../../common/Timestamp';
import cfg from '../../common/config';
import ExpirationDateTimePicker, { isExpirationInvalid } from './ExpirationDateTimePicker';
import TRNButton from '../../common/TRNButton';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import ClientCredentialsDialog from './ClientCredentialsDialog';
import { CLIENT_CREDENTIALS_DESCRIPTION } from './ServiceAccountForm';
import { GetConnections } from './ServiceAccountList';
import { ServiceAccountDetailsDeleteMutation } from './__generated__/ServiceAccountDetailsDeleteMutation.graphql';
import { ServiceAccountDetailsFragment_group$key } from './__generated__/ServiceAccountDetailsFragment_group.graphql';
import { ServiceAccountDetailsQuery } from './__generated__/ServiceAccountDetailsQuery.graphql';
import { ServiceAccountDetailsResetClientCredentialsMutation } from './__generated__/ServiceAccountDetailsResetClientCredentialsMutation.graphql';

interface Props {
    fragmentRef: ServiceAccountDetailsFragment_group$key
}

function ServiceAccountDetails(props: Props) {
    const { id } = useParams();
    const { enqueueSnackbar } = useSnackbar();
    const navigate = useNavigate();
    const [searchParams, setSearchParams] = useSearchParams();
    const [menuAnchorEl, setMenuAnchorEl] = useState<Element | null>(null);
    const [showDeleteConfirmationDialog, setShowDeleteConfirmationDialog] = useState<boolean>(false);

    const serviceAccountId = id as string;
    const validTabs = ['oidc', 'clientCredentials'];
    const tabParam = searchParams.get('tab');
    const tab = tabParam && validTabs.includes(tabParam) ? tabParam : 'oidc';

    const onTabChange = (_: React.SyntheticEvent, newValue: string) => {
        searchParams.set('tab', newValue);
        setSearchParams(searchParams, { replace: true });
    };

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
                clientCredentialsEnabled
                clientSecretExpiresAt
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

    const apiConfig = useContext(ApiConfigContext);
    const maxExpirationDays = apiConfig.serviceAccountClientSecretMaxExpirationDays;

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

    const [resetClientCredentialsCommit, resetClientCredentialsInFlight] = useMutation<ServiceAccountDetailsResetClientCredentialsMutation>(graphql`
        mutation ServiceAccountDetailsResetClientCredentialsMutation($input: ResetServiceAccountClientCredentialsInput!) {
            resetServiceAccountClientCredentials(input: $input) {
                serviceAccount {
                    id
                    clientCredentialsEnabled
                    clientSecretExpiresAt
                }
                clientSecret
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const [showResetClientCredentialsDialog, setShowResetClientCredentialsDialog] = useState(false);
    const [resetClientCredentialsExpiresAt, setResetClientCredentialsExpiresAt] = useState<Moment | null>(() => moment().add(maxExpirationDays, 'days'));
    const [clientCredentials, setClientCredentials] = useState<{ clientId: string; clientSecret: string } | null>(null);

    const onResetClientCredentials = () => {
        resetClientCredentialsCommit({
            variables: {
                input: {
                    id: serviceAccountId,
                    clientSecretExpiresAt: resetClientCredentialsExpiresAt?.toISOString()
                }
            },
            onCompleted: data => {
                setShowResetClientCredentialsDialog(false);
                setResetClientCredentialsExpiresAt(moment().add(maxExpirationDays, 'days'));
                if (data.resetServiceAccountClientCredentials.problems.length) {
                    enqueueSnackbar(data.resetServiceAccountClientCredentials.problems.map((problem: any) => problem.message).join('; '), { variant: 'warning' });
                } else if (data.resetServiceAccountClientCredentials.serviceAccount && data.resetServiceAccountClientCredentials.clientSecret) {
                    setClientCredentials({
                        clientId: data.resetServiceAccountClientCredentials.serviceAccount.id,
                        clientSecret: data.resetServiceAccountClientCredentials.clientSecret
                    });
                }
            },
            onError: error => {
                setShowResetClientCredentialsDialog(false);
                setResetClientCredentialsExpiresAt(moment().add(maxExpirationDays, 'days'));
                enqueueSnackbar(`Unexpected error occurred: ${error.message}`, { variant: 'error' });
            }
        });
    };

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

    const secretExpiresSoon = data.serviceAccount?.clientCredentialsEnabled &&
        data.serviceAccount.clientSecretExpiresAt &&
        moment(data.serviceAccount.clientSecretExpiresAt as moment.MomentInput).diff(moment(), 'days') <= 7;

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
                {secretExpiresSoon && (
                    <Alert severity="warning" variant="outlined" sx={{ mt: 2, mb: 2 }}>
                        Client secret expires {moment(data.serviceAccount.clientSecretExpiresAt as moment.MomentInput).fromNow()}. Reset the credentials to avoid authentication failures.
                    </Alert>
                )}
                <Box display="flex" justifyContent="space-between" marginBottom={2}>
                    <Box display="flex" alignItems="center">
                        <Avatar variant="rounded" sx={{ width: 32, height: 32, marginRight: 1, bgcolor: 'avatar.default' }}>
                            {data.serviceAccount.name[0].toUpperCase()}
                        </Avatar>
                        <Box>
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
                                {data.serviceAccount.clientCredentialsEnabled && (
                                    <MenuItem onClick={() => onMenuAction(() => setShowResetClientCredentialsDialog(true))}>
                                        Reset Client Credentials
                                    </MenuItem>
                                )}
                                <MenuItem onClick={() => onMenuAction(() => setShowDeleteConfirmationDialog(true))}>
                                    Delete Service Account
                                </MenuItem>
                            </Menu>
                        </Stack>
                    </Box>
                </Box>
                <Box sx={{ display: "flex", justifyContent: "space-between", border: 1, borderTopLeftRadius: 4, borderTopRightRadius: 4, borderColor: 'divider' }}>
                    <Tabs value={tab} onChange={onTabChange} aria-label="authentication methods">
                        <Tab label="OIDC Federation" value="oidc" />
                        <Tab label="Client Credentials" value="clientCredentials" />
                    </Tabs>
                </Box>
                <Box sx={{ border: 1, borderTop: 0, borderBottomLeftRadius: 4, borderBottomRightRadius: 4, borderColor: 'divider', padding: 2 }}>
                    {tab === 'oidc' && <Box>
                        <Typography variant="body2" color="textSecondary" sx={{ mb: 2 }}>
                            Tokens issued by trusted identity providers can authenticate to this service account if the bound claims match the token claims.
                        </Typography>
                        {data.serviceAccount.oidcTrustPolicies.length > 0 ? (
                            <Paper sx={{ padding: 1 }}>
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
                        ) : (
                            <Paper sx={{ padding: 2 }}>
                                <Typography variant="body2" color="textSecondary">
                                    No identity providers configured. Edit the service account to add OIDC trust policies.
                                </Typography>
                            </Paper>
                        )}
                    </Box>}
                    {tab === 'clientCredentials' && <Box>
                        <Typography variant="body2" color="textSecondary" sx={{ mb: 2 }}>
                            {CLIENT_CREDENTIALS_DESCRIPTION}
                        </Typography>
                        {data.serviceAccount.clientCredentialsEnabled ? (
                            <Paper sx={{ padding: 2 }}>
                                <Typography variant="body2" fontWeight="medium">Client ID</Typography>
                                <Box display="flex" alignItems="center" gap={0.5} mb={2}>
                                    <Typography variant="body2" color="textSecondary">{data.serviceAccount.id}</Typography>
                                    <CopyButton data={data.serviceAccount.id} toolTip="Click to copy" />
                                </Box>
                                <Typography variant="body2" fontWeight="medium">Token Endpoint (POST)</Typography>
                                <Box display="flex" alignItems="center" gap={0.5} mb={2}>
                                    <Typography variant="body2" color="textSecondary">{cfg.apiUrl}/v1/serviceaccounts/token</Typography>
                                    <CopyButton data={`${cfg.apiUrl}/v1/serviceaccounts/token`} toolTip="Click to copy" />
                                </Box>
                                <Typography variant="body2" fontWeight="medium">Secret Expires</Typography>
                                <Timestamp timestamp={data.serviceAccount.clientSecretExpiresAt as string} format="absolute" variant="body2" color="textSecondary" />
                            </Paper>
                        ) : (
                            <Paper sx={{ padding: 2 }}>
                                <Typography variant="body2" color="textSecondary">
                                    Client credentials are disabled. Edit the service account to enable this authentication method.
                                </Typography>
                            </Paper>
                        )}
                    </Box>}
                </Box>
                {showDeleteConfirmationDialog && (
                    <ConfirmationDialog
                        title="Delete Service Account"
                        confirmLabel="Delete"
                        confirmInProgress={commitInFlight}
                        onConfirm={() => onDeleteConfirmationDialogClosed(true)}
                        onClose={() => onDeleteConfirmationDialogClosed()}
                    >
                        Are you sure you want to delete service account <strong>{data.serviceAccount.resourcePath}</strong>?
                    </ConfirmationDialog>
                )}
                {showResetClientCredentialsDialog && (
                    <ConfirmationDialog
                        title="Reset Client Credentials"
                        confirmLabel="Reset"
                        confirmInProgress={resetClientCredentialsInFlight}
                        confirmDisabled={isExpirationInvalid(resetClientCredentialsExpiresAt, maxExpirationDays)}
                        onConfirm={onResetClientCredentials}
                        onClose={() => { setShowResetClientCredentialsDialog(false); setResetClientCredentialsExpiresAt(moment().add(maxExpirationDays, 'days')); }}
                    >
                        <Alert severity="warning" sx={{ mb: 2 }}>
                            This will invalidate the current client secret. Any applications using the current credentials will need to be updated.
                        </Alert>
                        <ExpirationDateTimePicker
                            label="New Secret Expiration (Optional)"
                            value={resetClientCredentialsExpiresAt}
                            onChange={setResetClientCredentialsExpiresAt}
                            maxExpirationDays={maxExpirationDays}
                            width="100%"
                        />
                    </ConfirmationDialog>
                )}
                {clientCredentials && (
                    <ClientCredentialsDialog
                        clientId={clientCredentials.clientId}
                        clientSecret={clientCredentials.clientSecret}
                        onClose={() => setClientCredentials(null)}
                    />
                )}
            </Box >
        );
    } else {
        return <Box display="flex" justifyContent="center" marginTop={4}>
            <Typography color="textSecondary">Service account with ID {serviceAccountId} not found</Typography>
        </Box>;
    }
}

export default ServiceAccountDetails
