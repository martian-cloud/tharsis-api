import {
    Alert,
    Box,
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    FormControl,
    InputAdornment,
    InputLabel,
    MenuItem,
    Paper,
    Select,
    TextField,
    Typography,
} from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { Moment } from 'moment';
import { useState } from 'react';
import { useLazyLoadQuery, useMutation } from 'react-relay/hooks';
import ActorDisplay from '../../common/ActorDisplay';
import CopyButton from '../../common/CopyButton';
import { MutationError } from '../../common/error';
import ExpirationDateTimePicker, { isExpirationInvalid } from '../../common/ExpirationDateTimePicker';
import NoResults from '../../common/NoResults';
import Timestamp from '../../common/Timestamp';
import { AdminAreaSCIMTokenSettingsMutation } from './__generated__/AdminAreaSCIMTokenSettingsMutation.graphql';
import { AdminAreaSCIMTokenSettingsQuery } from './__generated__/AdminAreaSCIMTokenSettingsQuery.graphql';

const query = graphql`
    query AdminAreaSCIMTokenSettingsQuery {
        scimToken {
            createdByActor {
                ...ActorDisplayFragment_actor
            }
            expiresAt
            metadata {
                createdAt
            }
        }
        config {
            scimTokenMaxExpirationDays
            oauthProviders {
                issuerUrl
            }
        }
    }
`;

function AdminAreaSCIMTokenSettings() {
    const data = useLazyLoadQuery<AdminAreaSCIMTokenSettingsQuery>(query, {}, { fetchPolicy: 'network-only' });
    const [selectedIssuerUrl, setSelectedIssuerUrl] = useState('');
    const [expiresAt, setExpiresAt] = useState<Moment | null>(null);
    const [confirmationOpen, setConfirmationOpen] = useState(false);
    const [scimToken, setSCIMToken] = useState(data.scimToken);
    const [tokenText, setTokenText] = useState<string | null>(null);
    const [error, setError] = useState<MutationError>();

    const [commit, isInFlight] = useMutation<AdminAreaSCIMTokenSettingsMutation>(graphql`
        mutation AdminAreaSCIMTokenSettingsMutation($input: CreateSCIMTokenInput!) {
            createSCIMToken(input: $input) {
                tokenText
                scimToken {
                    createdByActor {
                        ...ActorDisplayFragment_actor
                    }
                    expiresAt
                    metadata {
                        createdAt
                    }
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const closeConfirmationDialog = () => {
        setConfirmationOpen(false);
        setSelectedIssuerUrl('');
        setExpiresAt(null);
    };

    const createToken = () => {
        setError(undefined);

        commit({
            variables: {
                input: {
                    idpIssuerURL: selectedIssuerUrl,
                    expiresAt: expiresAt?.toISOString(),
                },
            },
            onCompleted: response => {
                if (response.createSCIMToken.problems.length) {
                    closeConfirmationDialog();
                    setError({
                        severity: 'warning',
                        message: response.createSCIMToken.problems.map(problem => problem.message).join('; '),
                    });
                    return;
                }

                if (!response.createSCIMToken.tokenText) {
                    closeConfirmationDialog();
                    setError({
                        severity: 'error',
                        message: 'The SCIM token could not be generated.',
                    });
                    return;
                }

                if (response.createSCIMToken.scimToken) {
                    setSCIMToken(response.createSCIMToken.scimToken);
                }
                setTokenText(response.createSCIMToken.tokenText);
            },
            onError: mutationError => {
                closeConfirmationDialog();
                setError({
                    severity: 'error',
                    message: `The SCIM token could not be generated: ${mutationError.message}`,
                });
            },
        });
    };

    const closeTokenDialog = () => {
        setConfirmationOpen(false);
        setTokenText(null);
        setSelectedIssuerUrl('');
        setExpiresAt(null);
    };

    const { oauthProviders, scimTokenMaxExpirationDays } = data.config;

    return (
        <Box>
            <Typography variant="h6" gutterBottom>SCIM Token Management</Typography>
            <Typography
                variant="body2"
                sx={{
                    color: "text.secondary",
                    mb: 2
                }}>
                Generate a SCIM token for a configured identity provider.
            </Typography>

            {error && <Alert severity={error.severity} sx={{ mb: 2 }}>{error.message}</Alert>}
            {scimToken ? (
                <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
                    <Typography variant="body2" sx={{ overflowWrap: 'anywhere' }}>
                        Created by: <strong><ActorDisplay fragmentRef={scimToken.createdByActor} variant="text" /></strong>
                    </Typography>
                    <Typography variant="body2">
                        Created at: <Timestamp timestamp={scimToken.metadata.createdAt} format="absolute" />
                    </Typography>
                    <Typography variant="body2">
                        Expires at: {scimToken.expiresAt ? <Timestamp timestamp={scimToken.expiresAt} format="absolute" /> : 'Never'}
                    </Typography>
                </Paper>
            ) : (
                <NoResults sx={{ mb: 2 }}>
                    No SCIM token has been generated.
                </NoResults>
            )}

            <Button
                variant="outlined"
                size="small"
                disabled={isInFlight}
                onClick={() => setConfirmationOpen(true)}
            >
                {scimToken ? 'Generate New SCIM Token' : 'Generate SCIM Token'}
            </Button>

            {confirmationOpen && (
                <Dialog
                    open
                    maxWidth="sm"
                    fullWidth
                    keepMounted={false}
                    onClose={() => {
                        if (!isInFlight && tokenText === null) {
                            closeConfirmationDialog();
                        }
                    }}
                >
                    <DialogTitle>{tokenText ? 'SCIM Token Generated' : 'Generate SCIM Token'}</DialogTitle>
                    <DialogContent dividers>
                        {tokenText ? (
                            <>
                                <Alert severity="warning" sx={{ mb: 2 }}>
                                    Copy this token now. It is shown only once and cannot be retrieved after this dialog is closed.
                                </Alert>
                                <TextField
                                    fullWidth
                                    size="small"
                                    label="SCIM Token"
                                    value={tokenText}
                                    slotProps={{
                                        input: {
                                            readOnly: true,
                                            endAdornment: (
                                                <InputAdornment position="end">
                                                    <CopyButton data={tokenText} toolTip="Copy token" />
                                                </InputAdornment>
                                            ),
                                        },
                                    }}
                                />
                            </>
                        ) : (
                            <>
                                {scimToken && (
                                    <Alert severity="warning" sx={{ mb: 2 }}>
                                        The existing SCIM token will be invalidated immediately.
                                    </Alert>
                                )}
                                {!oauthProviders.length ? (
                                    <Alert severity="warning">
                                        No identity providers are configured. Configure an identity provider before generating a SCIM token.
                                    </Alert>
                                ) : (
                                    <>
                                        <Typography variant="body2" sx={{ mb: 2 }}>
                                            Select an identity provider to generate a new SCIM token.
                                        </Typography>
                                        <FormControl fullWidth size="small">
                                            <InputLabel id="scim-idp-label">Identity Provider</InputLabel>
                                            <Select
                                                labelId="scim-idp-label"
                                                label="Identity Provider"
                                                value={selectedIssuerUrl}
                                                onChange={event => setSelectedIssuerUrl(event.target.value)}
                                            >
                                                {oauthProviders.map(provider => (
                                                    <MenuItem key={provider.issuerUrl} value={provider.issuerUrl}>
                                                        {provider.issuerUrl}
                                                    </MenuItem>
                                                ))}
                                            </Select>
                                        </FormControl>
                                        <Box sx={{ mt: 2 }}>
                                            <ExpirationDateTimePicker
                                                label="Token Expiration (Optional)"
                                                value={expiresAt}
                                                onChange={setExpiresAt}
                                                maxExpirationDays={scimTokenMaxExpirationDays}
                                                width="100%"
                                                emptyCaption={`The token never expires if left empty. Min: 1 day${scimTokenMaxExpirationDays ? `, max: ${scimTokenMaxExpirationDays} days` : ''}`}
                                            />
                                        </Box>
                                    </>
                                )}
                            </>
                        )}
                    </DialogContent>
                    <DialogActions>
                        {tokenText ? (
                            <Button onClick={closeTokenDialog}>Done</Button>
                        ) : (
                            <>
                                <Button color="inherit" disabled={isInFlight} onClick={closeConfirmationDialog}>
                                    Cancel
                                </Button>
                                <Button
                                    color="primary"
                                    disabled={!selectedIssuerUrl || isInFlight || isExpirationInvalid(expiresAt, scimTokenMaxExpirationDays)}
                                    loading={isInFlight}
                                    onClick={createToken}
                                >
                                    Generate Token
                                </Button>
                            </>
                        )}
                    </DialogActions>
                </Dialog>
            )}
        </Box>
    );
}

export default AdminAreaSCIMTokenSettings;
