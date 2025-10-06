import { useState } from 'react';
import { ArrowDropUp } from '@mui/icons-material';
import { default as ArrowDropDown, default as ArrowDropDownIcon } from '@mui/icons-material/ArrowDropDown';
import { LoadingButton } from '@mui/lab';
import { Avatar, Alert, Box, Button, ButtonGroup, Collapse, Dialog, DialogActions, DialogTitle, DialogContent, Divider, Menu, MenuItem, Link, Paper, Stack, styled, Typography } from '@mui/material'
import teal from '@mui/material/colors/teal';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import TRNButton from '../../common/TRNButton';
import { MutationError } from '../../common/error';
import { useFragment, useLazyLoadQuery, useMutation } from 'react-relay'
import { useNavigate, useParams } from 'react-router-dom';
import graphql from 'babel-plugin-relay/macro';
import moment from 'moment';
import { useSnackbar } from 'notistack';
import { GetConnections } from './VCSProviderList';
import { VCSProviderDetailsFragment_group$key } from './__generated__/VCSProviderDetailsFragment_group.graphql'
import { VCSProviderDetailsQuery } from './__generated__/VCSProviderDetailsQuery.graphql'
import { VCSProviderDetailsDeleteMutation } from './__generated__/VCSProviderDetailsDeleteMutation.graphql';
import { VCSProviderDetailsResetOAuthMutation } from './__generated__/VCSProviderDetailsResetOAuthMutation.graphql';

const CARD_PADDING = 3;

interface Props {
    fragmentRef: VCSProviderDetailsFragment_group$key
}

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

interface DialogProps {
    open: boolean
    onClose: (confirm?: boolean) => void
    keepMounted: boolean
    deleteInProgress?: boolean
    resetInProgress?: boolean
    vcsProviderPath?: string | undefined
    error?: MutationError
}

function DeleteConfirmationDialog(props: DialogProps) {
    const { vcsProviderPath, deleteInProgress, onClose, open, ...other } = props;
    return (
        <Dialog
            maxWidth="xs"
            open={open}
            {...other}
        >
            <DialogTitle>Delete VCS Provider</DialogTitle>
            <DialogContent dividers>
                Are you sure you want to delete VCS provider <strong>{vcsProviderPath}</strong>?
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

function ResetOAuthDialog(props: DialogProps) {
    const { open, onClose, resetInProgress, error, ...other } = props

    return (
        <Dialog
            maxWidth="xs"
            open={open}
            {...other}
        >
            <DialogTitle>Reset OAuth Token</DialogTitle>
            <DialogContent dividers>
                <Box sx={{ mt: 2, mb: 2 }}>
                    {error && <Alert sx={{ mt: 2, mb: 2 }} severity={error.severity}>
                        {error.message}
                    </Alert>}
                    <Typography>
                        Resetting the OAuth token will immediately generate a new authorization URL and redirect the browser to the VCS provider to finalize the OAuth flow.
                    </Typography>
                </Box>
            </DialogContent>
            <DialogActions>
                <Button color="inherit" onClick={() => onClose()}>
                    Cancel
                </Button>
                <LoadingButton color="primary" loading={resetInProgress} onClick={() => onClose(true)}>
                    Reset OAuth Token
                </LoadingButton>
            </DialogActions>
        </Dialog>
    )
}

function VCSProviderDetails(props: Props) {
    const { id } = useParams();
    const { enqueueSnackbar } = useSnackbar();
    const navigate = useNavigate();
    const [menuAnchorEl, setMenuAnchorEl] = useState<Element | null>(null);
    const [showDeleteConfirmationDialog, setShowDeleteConfirmationDialog] = useState<boolean>(false);
    const [showResetOAuthDialog, setShowResetOAuthDialog] = useState<boolean>(false)
    const [showMore, setShowMore] = useState(false);
    const [error, setError] = useState<MutationError>()

    const vcsProviderId = id as string;

    const group = useFragment<VCSProviderDetailsFragment_group$key>(
        graphql`
        fragment VCSProviderDetailsFragment_group on Group
        {
            id
            fullPath
        }
        `, props.fragmentRef)

    const data = useLazyLoadQuery<VCSProviderDetailsQuery>(graphql`
        query VCSProviderDetailsQuery($id: String!) {
            node(id: $id) {
                ... on VCSProvider {
                    id
                    name
                    createdBy
                    description
                    type
                    url
                    resourcePath
                    autoCreateWebhooks
                    metadata {
                        createdAt
                        trn
                    }
                }
            }
        }
    `, { id: vcsProviderId });

    const [commit, commitInFlight] = useMutation<VCSProviderDetailsDeleteMutation>(graphql`
        mutation VCSProviderDetailsDeleteMutation($input: DeleteVCSProviderInput!, $connections: [ID!]!) {
            deleteVCSProvider(input: $input) {
                vcsProvider {
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
                        id: vcsProviderId
                    },
                    connections: GetConnections(group.id),
                },
                onCompleted: data => {
                    setShowDeleteConfirmationDialog(false);

                    if (data.deleteVCSProvider.problems.length) {
                        enqueueSnackbar(data.deleteVCSProvider.problems.map((problem: any) =>   problem.  message).join('; '), { variant: 'warning' });
                    } else {
                        navigate(`..`);
                    }
                },
                onError: error => {
                    setShowDeleteConfirmationDialog(false);
                    enqueueSnackbar(`Unexpected error occurred: ${error.message}`, { variant:   'error' });
                }
            });
        } else {
            setShowDeleteConfirmationDialog(false);
        }
    };

    const [commitResetOAuth, commitResetOAuthInFlight] = useMutation<VCSProviderDetailsResetOAuthMutation>(graphql`
        mutation VCSProviderDetailsResetOAuthMutation($input: ResetVCSProviderOAuthTokenInput!) {
            resetVCSProviderOAuthToken(input: $input) {
                oAuthAuthorizationUrl
                problems {
                    message
                    field
                    type
                }
            }
        }
    `)

    const onResetOAuthDialogClosed = (confirm?: boolean) => {
        if (confirm) {
            commitResetOAuth({
                variables: {
                    input: {
                        providerId: vcsProviderId
                    }
                },
            onCompleted: data => {
                if (data.resetVCSProviderOAuthToken.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.resetVCSProviderOAuthToken.problems.map((problem: any) => problem.message).join('; ')
                    });
                } else if (!data.resetVCSProviderOAuthToken.oAuthAuthorizationUrl) {
                    setError({
                        severity: 'error',
                        message: "Unexpected error occurred"
                    });
                }
                else {
                    enqueueSnackbar('OAuth token reset', { variant: 'success' })
                    window.open(data.resetVCSProviderOAuthToken.oAuthAuthorizationUrl, '_blank')
                    setShowResetOAuthDialog(false);
                }
            },
            onError: error => {
                setError({
                    severity: 'error',
                    message: `Unexpected error occurred: ${error.message}`
                });
            }
        });
        } else {
            setShowResetOAuthDialog(false);
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

    if (data.node && id) {

        const vcsProvider = data.node as any;

        return (
            <Box>
                <NamespaceBreadcrumbs
                    namespacePath={group.fullPath}
                    childRoutes={[
                        { title: "vcs providers", path: 'vcs_providers' },
                        { title: vcsProvider.name, path: id }
                    ]}
                />
                <Paper variant="outlined" sx={{ marginTop: 3, padding: CARD_PADDING }}>
                    <Box display="flex" justifyContent="space-between">
                        <Box display="flex" alignItems="center">
                            <Avatar variant="rounded" sx={{ width: 32, height: 32, marginRight: 1, bgcolor: teal[200] }}>
                                {vcsProvider.name[0].toUpperCase()}
                            </Avatar>
                            <Box marginLeft={1}>
                                <Box display="flex" alignItems="center">
                                    <Typography variant="h5" sx={{ marginRight: 1 }}>{vcsProvider.name}</Typography>
                                </Box>
                                <Typography color="textSecondary">{vcsProvider.description}</Typography>
                            </Box>
                        </Box>
                        <Box>
                            <Stack direction="row" spacing={1} >
                                <TRNButton trn={vcsProvider.metadata.trn} />
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
                                    id="vcs-provider-more-options-menu"
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
                                    <MenuItem onClick={() => navigate('edit_oauth_credentials')}>
                                        Edit OAuth Credentials
                                    </MenuItem>
                                    <MenuItem onClick={() => onMenuAction(() =>
                                        setShowResetOAuthDialog(true))}>
                                        Reset OAuth Token
                                    </MenuItem>
                                    <MenuItem onClick={() => onMenuAction(() => setShowDeleteConfirmationDialog(true))}>
                                        Delete VCS Provider
                                    </MenuItem>
                                </Menu>
                            </Stack>
                        </Box>
                    </Box>
                    <Divider sx={{ opacity: 0.6, marginTop: 2, marginBottom: 2, marginLeft: -CARD_PADDING, marginRight: -CARD_PADDING }} />
                    <Box>
                        <Box>
                            <FieldLabel>Type</FieldLabel>
                            <FieldValue>{vcsProvider.type === 'github' ? 'GitHub' : 'GitLab'}</FieldValue>
                            <FieldLabel>URL</FieldLabel>
                            <FieldValue>{vcsProvider.url}</FieldValue>
                            <FieldLabel>Automatically create webhooks?</FieldLabel>
                            <FieldValue>{vcsProvider.autoCreateWebhooks ? 'Yes' : 'No'}</FieldValue>
                        </Box>
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
                                        Created {moment(vcsProvider.metadata.createdAt as moment.MomentInput).fromNow()} by {vcsProvider.createdBy}
                                    </Typography>
                                </Box>
                            </Collapse>
                        </Box>
                    </Box>
                </Paper >
                <DeleteConfirmationDialog
                    vcsProviderPath={vcsProvider.resourcePath}
                    keepMounted
                    deleteInProgress={commitInFlight}
                    open={showDeleteConfirmationDialog}
                    onClose={onDeleteConfirmationDialogClosed}
                />
                <ResetOAuthDialog
                    open={showResetOAuthDialog}
                    keepMounted
                    onClose={onResetOAuthDialogClosed}
                    resetInProgress={commitResetOAuthInFlight}
                    error={error}
                />
            </Box>
        );
    }
    else {
        return <Box display="flex" justifyContent="center" marginTop={4}>
            <Typography color="textSecondary">VCS Provider with ID {vcsProviderId} not found</Typography>
        </Box>;
    }
}

export default VCSProviderDetails
