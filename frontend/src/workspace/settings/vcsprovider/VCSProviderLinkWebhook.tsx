import { Alert, AlertTitle, Box, Button, Chip, Paper, Stack, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import { useState } from 'react';
import { useFragment, useMutation } from 'react-relay';
import ConfirmationDialog from '../../../common/ConfirmationDialog';
import { MutationError } from '../../../common/error';
import { ManualWebhookDetails } from './ManualWebhookDialog';
import { manualWebhookDetailsLabel, providerLabel } from './utils';
import { VCSProviderLinkWebhookFragment_link$key } from './__generated__/VCSProviderLinkWebhookFragment_link.graphql';
import { VCSProviderLinkWebhookResetMutation } from './__generated__/VCSProviderLinkWebhookResetMutation.graphql';

interface Props {
    fragmentRef: VCSProviderLinkWebhookFragment_link$key
    onShowWebhookDetails: (details: ManualWebhookDetails) => void
}

function VCSProviderLinkWebhook({ fragmentRef, onShowWebhookDetails }: Props) {
    const { enqueueSnackbar } = useSnackbar();

    const data = useFragment<VCSProviderLinkWebhookFragment_link$key>(
        graphql`
        fragment VCSProviderLinkWebhookFragment_link on WorkspaceVCSProviderLink
        {
            id
            repositoryPath
            webhookDisabled
            usesLegacyWebhookToken
            vcsProvider {
                type
                autoCreateWebhooks
            }
        }
        `, fragmentRef
    );

    const [commit, isInFlight] = useMutation<VCSProviderLinkWebhookResetMutation>(graphql`
        mutation VCSProviderLinkWebhookResetMutation($input: ResetWorkspaceVCSProviderLinkWebhookTokenInput!) {
            resetWorkspaceVCSProviderLinkWebhookToken(input: $input) {
                vcsProviderLink {
                    id
                    usesLegacyWebhookToken
                }
                webhookToken
                webhookUrl
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const [showResetDialog, setShowResetDialog] = useState(false);
    const [error, setError] = useState<MutationError>();

    const auto = data.vcsProvider.autoCreateWebhooks;
    const provider = providerLabel(data.vcsProvider.type);
    const repo = data.repositoryPath;
    const outOfDate = !data.webhookDisabled && data.usesLegacyWebhookToken;

    const onReset = () => {
        commit({
            variables: { input: { id: data.id } },
            onCompleted: response => {
                setShowResetDialog(false);

                const payload = response.resetWorkspaceVCSProviderLinkWebhookToken;
                if (payload.problems.length) {
                    setError({ severity: 'warning', message: payload.problems.map(problem => problem.message).join('; ') });
                    return;
                }

                if (!payload.vcsProviderLink) {
                    setError({ severity: 'error', message: 'Unexpected error occurred' });
                    return;
                }

                setError(undefined);
                // A webhook URL is only returned when the webhook has to be updated by hand.
                if (payload.webhookUrl) {
                    enqueueSnackbar(`Webhook reset. The old webhook stopped working. Add the new ${manualWebhookDetailsLabel(data.vcsProvider.type)} in ${provider}, or runs won't start.`, { variant: 'success' });
                    onShowWebhookDetails({
                        context: 'reset',
                        url: payload.webhookUrl,
                        token: payload.webhookToken,
                        providerType: data.vcsProvider.type,
                        repositoryPath: repo
                    });
                } else {
                    enqueueSnackbar(`Webhook reset. Tharsis generated a new secret and updated the webhook in ${repo}.`, { variant: 'success' });
                }
            },
            onError: error => {
                setShowResetDialog(false);
                setError({ severity: 'error', message: `Unexpected error occurred: ${error.message}` });
            }
        });
    };

    return (
        <Paper sx={{ p: 3 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 1, mb: 2 }}>
                <Typography variant="h6">Webhook</Typography>
                <Stack direction="row" spacing={1}>
                    {auto
                        ? <Chip size="small" variant="outlined" color="success" label="Created by Tharsis" />
                        : <Chip size="small" variant="outlined" color="info" label="Set up manually" />}
                    {data.webhookDisabled && <Chip size="small" variant="outlined" label="Disabled" />}
                </Stack>
            </Box>
            <Typography sx={{ mb: 2 }}>
                {auto
                    ? `Tharsis created this webhook in ${repo}. It sends push and merge request events to Tharsis.`
                    : `You added this webhook to ${repo} yourself. It sends push and merge request events to Tharsis.`}
            </Typography>
            {data.webhookDisabled && (
                <Alert severity="info" sx={{ mb: 2 }}>
                    <AlertTitle>Events are ignored</AlertTitle>
                    Pushes and merge requests in {repo} won't start runs. Turn on "Webhook Push Events" in link settings and save to use it again.
                </Alert>
            )}
            {outOfDate && (
                <Alert severity="warning" sx={{ mb: 2 }}>
                    <AlertTitle>Webhook is out of date</AlertTitle>
                    This webhook uses a legacy token, so runs may stop starting at any time. Reset it to replace the token{auto ? '' : ` and paste the new one into ${provider}`}.
                </Alert>
            )}
            {error && (
                <Alert severity={error.severity} sx={{ mb: 2 }}>
                    {error.message}
                </Alert>
            )}
            {!data.webhookDisabled && <>
                <Button size="small" variant="outlined" color="secondary" onClick={() => setShowResetDialog(true)}>Reset webhook</Button>
                <Typography variant="body2" color="textSecondary" sx={{ mt: 1 }}>
                    {auto
                        ? `Tharsis generates a new secret and updates the webhook in ${repo}.`
                        : `The current webhook stops working right away. You'll get a new ${manualWebhookDetailsLabel(data.vcsProvider.type)} to paste into ${repo}.`}
                </Typography>
            </>}
            {showResetDialog && (
                <ConfirmationDialog
                    title="Reset webhook?"
                    confirmLabel="Reset webhook"
                    confirmColor={auto ? 'primary' : 'error'}
                    confirmInProgress={isInFlight}
                    onConfirm={onReset}
                    onClose={() => setShowResetDialog(false)}
                >
                    <Typography>
                        {auto
                            ? `Tharsis will generate a new secret and update the webhook in ${repo}. Runs keep starting and nothing else changes.`
                            : `The current webhook stops working right away. Tharsis will show a new ${manualWebhookDetailsLabel(data.vcsProvider.type)} to paste into the webhook settings of ${repo}.`}
                    </Typography>
                    {!auto && (
                        <Alert severity="warning" sx={{ mt: 2 }}>
                            Pushes and merge requests won't start runs until you update the webhook in {provider}.
                        </Alert>
                    )}
                </ConfirmationDialog>
            )}
        </Paper>
    );
}

export default VCSProviderLinkWebhook;
