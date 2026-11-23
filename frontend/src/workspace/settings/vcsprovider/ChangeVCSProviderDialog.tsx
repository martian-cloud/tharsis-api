import { Alert, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import { useState } from 'react';
import { useMutation } from 'react-relay';
import ConfirmationDialog from '../../../common/ConfirmationDialog';
import { MutationError } from '../../../common/error';
import { ManualWebhookDetails } from './ManualWebhookDialog';
import { NewMutation } from './NewVCSProviderLink';
import VCSProviderAutocomplete, { VCSProviderOption } from './VCSProviderAutocomplete';
import { VCSFormData } from './VCSProviderLinkForm';
import { providerHelperText, providerLabel } from './utils';
import { ChangeVCSProviderDialogDeleteMutation } from './__generated__/ChangeVCSProviderDialogDeleteMutation.graphql';
import { NewVCSProviderLinkMutation } from './__generated__/NewVCSProviderLinkMutation.graphql';

// ChangeVCSProviderLink is a copy of the link taken when the dialog opens: changing the provider deletes the
// link before creating its replacement, so the dialog can't read the link from the store while it works.
export interface ChangeVCSProviderLink extends VCSFormData {
    id: string
    providerId: string
    providerName: string
    providerAutoCreateWebhooks: boolean
}

interface Props {
    link: ChangeVCSProviderLink
    workspacePath: string
    onClose: () => void
    onShowWebhookDetails: (details: ManualWebhookDetails) => void
}

function ChangeVCSProviderDialog({ link, workspacePath, onClose, onShowWebhookDetails }: Props) {
    const { enqueueSnackbar } = useSnackbar();

    const [provider, setProvider] = useState<VCSProviderOption | null>(null);
    const [error, setError] = useState<MutationError>();
    // Set once the old link is gone, so trying again only creates the replacement.
    const [oldLinkDeleted, setOldLinkDeleted] = useState(false);

    const [commitDelete, deleteInFlight] = useMutation<ChangeVCSProviderDialogDeleteMutation>(graphql`
        mutation ChangeVCSProviderDialogDeleteMutation($input: DeleteWorkspaceVCSProviderLinkInput!) {
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

    const [commitCreate, createInFlight] = useMutation<NewVCSProviderLinkMutation>(NewMutation);

    const onCreate = (newProvider: VCSProviderOption) => {
        commitCreate({
            variables: {
                input: {
                    workspacePath,
                    providerId: newProvider.id,
                    repositoryPath: link.repositoryPath,
                    branch: link.branch,
                    moduleDirectory: link.moduleDirectory,
                    tagRegex: link.tagRegex,
                    globPatterns: link.globPatterns,
                    autoSpeculativePlan: link.autoSpeculativePlan,
                    webhookDisabled: link.webhookDisabled
                }
            },
            onCompleted: data => {
                const payload = data.createWorkspaceVCSProviderLink;
                if (payload.problems.length || !payload.vcsProviderLink) {
                    setError(payload.problems.length
                        ? { severity: 'warning', message: payload.problems.map(problem => problem.message).join('; ') }
                        : { severity: 'error', message: 'Unexpected error occurred' });
                    return;
                }

                const oldWebhook = link.providerAutoCreateWebhooks
                    ? 'Tharsis deleted the old webhook.'
                    : `Delete the old webhook from ${link.repositoryPath} by hand.`;

                if (payload.webhookUrl) {
                    enqueueSnackbar(`Provider changed. ${oldWebhook} Add the new one in ${providerLabel(newProvider.type)} to start runs.`, { variant: 'success' });
                    onShowWebhookDetails({
                        context: 'provider_change',
                        url: payload.webhookUrl,
                        token: payload.webhookToken,
                        providerType: newProvider.type,
                        repositoryPath: link.repositoryPath
                    });
                } else if (newProvider.autoCreateWebhooks && !link.webhookDisabled) {
                    enqueueSnackbar(`Provider changed. ${oldWebhook} Tharsis created a new one in ${link.repositoryPath}.`, { variant: 'success' });
                } else {
                    enqueueSnackbar(`Provider changed. ${oldWebhook}`, { variant: 'success' });
                }

                onClose();
            },
            onError: error => {
                setError({ severity: 'error', message: `Unexpected error occurred: ${error.message}` });
            }
        });
    };

    const onConfirm = () => {
        if (!provider) {
            return;
        }

        if (oldLinkDeleted) {
            onCreate(provider);
            return;
        }

        commitDelete({
            variables: { input: { id: link.id } },
            onCompleted: data => {
                const problems = data.deleteWorkspaceVCSProviderLink.problems;
                if (problems.length) {
                    setError({ severity: 'warning', message: problems.map(problem => problem.message).join('; ') });
                    return;
                }

                setOldLinkDeleted(true);
                onCreate(provider);
            },
            onError: error => {
                setError({ severity: 'error', message: `Unexpected error occurred: ${error.message}` });
            }
        });
    };

    return (
        <ConfirmationDialog
            title="Change VCS provider?"
            maxWidth="sm"
            confirmLabel="Change provider"
            confirmDisabled={!provider}
            confirmInProgress={deleteInFlight || createInFlight}
            onConfirm={onConfirm}
            // Closing mid-change would hide the outcome of a link that may already be deleted.
            onClose={() => !deleteInFlight && !createInFlight && onClose()}
        >
            {error && (
                <Alert severity={error.severity} sx={{ mb: 2 }}>
                    {error.message}
                </Alert>
            )}
            {!oldLinkDeleted && (
                <Typography sx={{ mb: 3 }}>
                    This replaces the link. {link.providerAutoCreateWebhooks
                        ? `Tharsis will delete the webhook in ${link.repositoryPath} and set one up through the new provider.`
                        : `The webhook you added to ${link.repositoryPath} stays until you delete it, and a new one is set up through the new provider.`} Your repository, branch and other settings stay as they are.
                </Typography>
            )}
            {oldLinkDeleted && (
                <Typography sx={{ mb: 3 }}>
                    The old link was removed. Try again to create the link with the new provider.
                </Typography>
            )}
            <VCSProviderAutocomplete
                path={workspacePath}
                label="New provider"
                value={provider}
                onSelected={setProvider}
                excludeId={link.providerId}
                helperText={providerHelperText(provider, `Choose the provider that will replace ${link.providerName}.`)}
            />
        </ConfirmationDialog>
    );
}

export default ChangeVCSProviderDialog;
