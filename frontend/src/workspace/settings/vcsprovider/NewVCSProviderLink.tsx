import { Alert, Box, Button, Paper, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import { useState } from 'react';
import { GraphQLTaggedNode, useMutation } from 'react-relay';
import { MutationError } from '../../../common/error';
import { ManualWebhookDetails } from './ManualWebhookDialog';
import VCSProviderAutocomplete, { VCSProviderOption } from './VCSProviderAutocomplete';
import VCSProviderLinkForm, { EMPTY_VCS_FORM_DATA, VCSFormData } from './VCSProviderLinkForm';
import { providerHelperText } from './utils';
import { NewVCSProviderLinkMutation } from './__generated__/NewVCSProviderLinkMutation.graphql';

interface Props {
    workspacePath: string
    onLinked: () => void
    onCancel: () => void
    onShowWebhookDetails: (details: ManualWebhookDetails) => void
}

export const NewMutation: GraphQLTaggedNode =
    graphql`
    mutation NewVCSProviderLinkMutation($input: CreateWorkspaceVCSProviderLinkInput!) {
        createWorkspaceVCSProviderLink(input: $input) {
            vcsProviderLink {
                id
                repositoryPath
                vcsProvider {
                    type
                    autoCreateWebhooks
                }
                workspace {
                    id
                    workspaceVcsProviderLink {
                        id
                        ...EditVCSProviderLinkFragment_link
                        ...VCSProviderLinkWebhookFragment_link
                        ...VCSProviderLinkDangerZoneFragment_link
                    }
                }
            }
            webhookToken
            webhookUrl
            problems {
                message
                field
                type
            }
        }
    }`

function NewVCSProviderLink({ workspacePath, onLinked, onCancel, onShowWebhookDetails }: Props) {
    const { enqueueSnackbar } = useSnackbar();

    const [provider, setProvider] = useState<VCSProviderOption | null>(null);
    const [formData, setFormData] = useState<VCSFormData>(EMPTY_VCS_FORM_DATA);
    const [submitted, setSubmitted] = useState(false);
    const [error, setError] = useState<MutationError>();

    const [commit, isInFlight] = useMutation<NewVCSProviderLinkMutation>(NewMutation);

    const providerMissing = submitted && !provider;
    const repositoryPathMissing = submitted && formData.repositoryPath.trim() === '';

    const onLink = () => {
        setSubmitted(true);
        if (!provider || formData.repositoryPath.trim() === '') {
            return;
        }

        commit({
            variables: {
                input: {
                    workspacePath,
                    providerId: provider.id,
                    repositoryPath: formData.repositoryPath.trim(),
                    branch: formData.branch,
                    moduleDirectory: formData.moduleDirectory,
                    tagRegex: formData.tagRegex,
                    globPatterns: formData.globPatterns,
                    autoSpeculativePlan: formData.autoSpeculativePlan,
                    webhookDisabled: formData.webhookDisabled
                },
            },
            onCompleted: data => {
                const payload = data.createWorkspaceVCSProviderLink;
                if (payload.problems.length) {
                    setError({ severity: 'warning', message: payload.problems.map(problem => problem.message).join('; ') });
                    return;
                }

                const link = payload.vcsProviderLink;
                if (!link) {
                    setError({ severity: 'error', message: 'Unexpected error occurred' });
                    return;
                }

                // A webhook URL is only returned when the webhook has to be added by hand.
                if (payload.webhookUrl) {
                    enqueueSnackbar(`Repository linked. Add the webhook to ${link.repositoryPath} to start runs.`, { variant: 'success' });
                    onShowWebhookDetails({
                        context: 'new',
                        url: payload.webhookUrl,
                        token: payload.webhookToken,
                        providerType: link.vcsProvider.type,
                        repositoryPath: link.repositoryPath
                    });
                } else {
                    enqueueSnackbar(`Repository linked. Tharsis created the webhook in ${link.repositoryPath}.`, { variant: 'success' });
                }

                onLinked();
            },
            onError: error => {
                setError({ severity: 'error', message: `Unexpected error occurred: ${error.message}` });
            }
        });
    };

    return (
        <Paper sx={{ p: 3, mt: 2 }}>
            <Typography variant="h6" sx={{ mb: 3 }}>Link a repository</Typography>
            {error && (
                <Alert severity={error.severity} sx={{ mb: 3 }}>
                    {error.message}
                </Alert>
            )}
            <Box sx={{ mb: 2 }}>
                <VCSProviderAutocomplete
                    path={workspacePath}
                    value={provider}
                    onSelected={setProvider}
                    error={providerMissing}
                    helperText={providerMissing ? 'Choose a provider.' : providerHelperText(provider, 'Pick the GitHub or GitLab connection that can read this repository.')}
                />
            </Box>
            <VCSProviderLinkForm
                data={formData}
                onChange={setFormData}
                repositoryPathError={repositoryPathMissing}
            />
            <Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: 1, mt: 3 }}>
                <Button size="small" color="inherit" onClick={onCancel} disabled={isInFlight}>Cancel</Button>
                <Button size="small" variant="outlined" color="primary" loading={isInFlight} onClick={onLink}>Link repository</Button>
            </Box>
        </Paper>
    );
}

export default NewVCSProviderLink
