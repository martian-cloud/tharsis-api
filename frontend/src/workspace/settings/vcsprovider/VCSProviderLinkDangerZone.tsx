import { Alert, AlertTitle, Box, Button, Checkbox, FormControlLabel, Paper, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import { useState } from 'react';
import { useFragment, useMutation } from 'react-relay';
import ConfirmationDialog from '../../../common/ConfirmationDialog';
import { VCSProviderLinkDangerZoneDeleteMutation } from './__generated__/VCSProviderLinkDangerZoneDeleteMutation.graphql';
import { VCSProviderLinkDangerZoneFragment_link$key } from './__generated__/VCSProviderLinkDangerZoneFragment_link.graphql';

interface Props {
    fragmentRef: VCSProviderLinkDangerZoneFragment_link$key
}

function VCSProviderLinkDangerZone({ fragmentRef }: Props) {
    const { enqueueSnackbar } = useSnackbar();

    const data = useFragment<VCSProviderLinkDangerZoneFragment_link$key>(
        graphql`
        fragment VCSProviderLinkDangerZoneFragment_link on WorkspaceVCSProviderLink
        {
            id
            repositoryPath
            vcsProvider {
                autoCreateWebhooks
            }
        }
        `, fragmentRef
    );

    const [commit, isInFlight] = useMutation<VCSProviderLinkDangerZoneDeleteMutation>(graphql`
        mutation VCSProviderLinkDangerZoneDeleteMutation($input: DeleteWorkspaceVCSProviderLinkInput!) {
            deleteWorkspaceVCSProviderLink(input: $input) {
                vcsProviderLink {
                    workspace {
                        id
                        workspaceVcsProviderLink {
                            id
                        }
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

    const [showDialog, setShowDialog] = useState(false);
    const [force, setForce] = useState(false);

    const repo = data.repositoryPath;
    // Tharsis only removes webhooks it created; a manually added one is left for the user to delete.
    const auto = data.vcsProvider.autoCreateWebhooks;

    const openDialog = () => {
        setForce(false);
        setShowDialog(true);
    };

    const onUnlink = () => {
        commit({
            variables: { input: { id: data.id, force } },
            onCompleted: response => {
                setShowDialog(false);

                const problems = response.deleteWorkspaceVCSProviderLink.problems;
                if (problems.length) {
                    enqueueSnackbar(problems.map(problem => problem.message).join('; '), { variant: 'warning' });
                    return;
                }

                if (force) {
                    enqueueSnackbar(`Repository unlinked, but the webhook may still exist. To stop events being sent, delete it by hand in ${repo} under Settings → Webhooks.`, { variant: 'warning' });
                } else if (auto) {
                    enqueueSnackbar(`Repository unlinked. Tharsis deleted the link and removed the webhook from ${repo}.`, { variant: 'success' });
                } else {
                    enqueueSnackbar(`Repository unlinked. Delete the webhook by hand in ${repo} under Settings → Webhooks.`, { variant: 'success' });
                }
            },
            onError: error => {
                setShowDialog(false);
                enqueueSnackbar(`Unexpected error occurred: ${error.message}`, { variant: 'error' });
            }
        });
    };

    return (
        <Paper variant="outlined" sx={{ p: 3 }}>
            <Typography variant="h6" color="error" sx={{ mb: 2 }}>Danger zone</Typography>
            <Box
                sx={{
                    display: 'flex',
                    flexDirection: { xs: 'column', sm: 'row' },
                    alignItems: { sm: 'center' },
                    justifyContent: 'space-between',
                    gap: 2
                }}
            >
                <Box sx={{ minWidth: 0 }}>
                    <Typography sx={{ fontWeight: 500 }}>Unlink repository</Typography>
                    <Typography variant="body2" color="textSecondary">
                        {auto
                            ? `Deletes the link and tries to remove the webhook from ${repo}.`
                            : `Deletes the link. Remove the webhook from ${repo} by hand afterwards.`} Pushes and merge requests stop starting runs. Existing runs aren't affected.
                    </Typography>
                </Box>
                <Button size="small" variant="outlined" color="error" sx={{ flexShrink: 0 }} onClick={openDialog}>
                    Unlink repository
                </Button>
            </Box>
            {showDialog && (
                <ConfirmationDialog
                    title="Unlink repository"
                    maxWidth="sm"
                    confirmLabel="Unlink"
                    typedConfirmation={force ? { label: 'Enter the following to confirm:', value: repo } : undefined}
                    confirmInProgress={isInFlight}
                    onConfirm={onUnlink}
                    onClose={() => setShowDialog(false)}
                >
                    <Alert severity="warning" sx={{ mb: 2 }}>
                        <AlertTitle>Warning</AlertTitle>
                        {auto
                            ? `Tharsis will delete this link and try to remove the webhook from ${repo}.`
                            : `Tharsis will delete this link. The webhook you added to ${repo} stays until you remove it.`} Pushes and merge requests will stop starting runs. Existing runs aren't affected and your code isn't changed.
                        {force && ` With force, the link is deleted even if the webhook can't be removed. Delete it by hand under Settings → Webhooks, or events will keep being sent to Tharsis.`}
                    </Alert>
                    {auto && (
                        <FormControlLabel
                            sx={{ mb: 1 }}
                            control={<Checkbox checked={force} onChange={event => setForce(event.target.checked)} />}
                            label="Force: unlink even if the webhook can't be removed"
                        />
                    )}
                </ConfirmationDialog>
            )}
        </Paper>
    );
}

export default VCSProviderLinkDangerZone;
