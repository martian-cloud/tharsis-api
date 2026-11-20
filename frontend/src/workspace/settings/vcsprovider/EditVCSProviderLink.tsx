import { Alert, Box, Button, Chip, Link, Paper, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import { useMemo, useState } from 'react';
import { useFragment, useMutation } from 'react-relay';
import { Link as RouterLink } from 'react-router-dom';
import { MutationError } from '../../../common/error';
import VCSProviderTypeChip from '../../../groups/vcsprovider/VCSProviderTypeChip';
import { ChangeVCSProviderLink } from './ChangeVCSProviderDialog';
import VCSProviderLinkForm, { isFormDataEqual, mapLinkToFormData, VCSFormData } from './VCSProviderLinkForm';
import { EditVCSProviderLinkFragment_link$key } from './__generated__/EditVCSProviderLinkFragment_link.graphql';
import { EditVCSProviderLinkMutation } from './__generated__/EditVCSProviderLinkMutation.graphql';

interface Props {
    fragmentRef: EditVCSProviderLinkFragment_link$key
    onChangeProvider: (link: ChangeVCSProviderLink) => void
}

function EditVCSProviderLink({ fragmentRef, onChangeProvider }: Props) {
    const { enqueueSnackbar } = useSnackbar();

    const data = useFragment<EditVCSProviderLinkFragment_link$key>(
        graphql`
        fragment EditVCSProviderLinkFragment_link on WorkspaceVCSProviderLink
        {
            id
            repositoryPath
            branch
            moduleDirectory
            tagRegex
            globPatterns
            autoSpeculativePlan
            webhookDisabled
            vcsProvider {
                id
                name
                groupPath
                type
                autoCreateWebhooks
            }
        }
        `, fragmentRef
    );

    const [commit, isInFlight] = useMutation<EditVCSProviderLinkMutation>(
        graphql`
        mutation EditVCSProviderLinkMutation($input: UpdateWorkspaceVCSProviderLinkInput!) {
            updateWorkspaceVCSProviderLink(input: $input) {
                vcsProviderLink {
                    id
                    ...EditVCSProviderLinkFragment_link
                    ...VCSProviderLinkWebhookFragment_link
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const savedFormData = useMemo(() => mapLinkToFormData(data), [data]);
    const [formData, setFormData] = useState<VCSFormData>(savedFormData);
    const [error, setError] = useState<MutationError>();

    const dirty = !isFormDataEqual(formData, savedFormData);

    const onSave = () => {
        commit({
            variables: {
                input: {
                    id: data.id,
                    branch: formData.branch,
                    moduleDirectory: formData.moduleDirectory,
                    tagRegex: formData.tagRegex,
                    globPatterns: formData.globPatterns,
                    autoSpeculativePlan: formData.autoSpeculativePlan,
                    webhookDisabled: formData.webhookDisabled
                }
            },
            onCompleted: response => {
                const payload = response.updateWorkspaceVCSProviderLink;
                if (payload.problems.length) {
                    setError({ severity: 'warning', message: payload.problems.map(problem => problem.message).join('; ') });
                } else if (!payload.vcsProviderLink) {
                    setError({ severity: 'error', message: 'Unexpected error occurred' });
                } else {
                    setError(undefined);
                    enqueueSnackbar('Link settings saved.', { variant: 'success' });
                }
            },
            onError: error => {
                setError({ severity: 'error', message: `Unexpected error occurred: ${error.message}` });
            }
        });
    };

    const onDiscard = () => {
        setFormData(savedFormData);
        setError(undefined);
    };

    return (
        <Paper sx={{ p: 3 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 2, mb: 3 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, minWidth: 0 }}>
                    <VCSProviderTypeChip type={data.vcsProvider.type} />
                    <Box sx={{ minWidth: 0 }}>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                            <Typography variant="h6">Link settings</Typography>
                            {dirty && <Chip size="small" variant="outlined" color="warning" label="Unsaved changes" />}
                        </Box>
                        <Typography variant="body2" color="textSecondary" noWrap>
                            Provider: <Link
                                component={RouterLink}
                                to={`/groups/${data.vcsProvider.groupPath}/-/vcs_providers/${data.vcsProvider.id}`}
                                underline="hover"
                            >
                                {data.vcsProvider.name}
                            </Link>
                        </Typography>
                    </Box>
                </Box>
                <Button
                    size="small"
                    variant="outlined"
                    color="secondary"
                    sx={{ flexShrink: 0 }}
                    // The replacement link copies the saved settings, so unsaved edits must be saved or discarded first.
                    disabled={dirty}
                    onClick={() => onChangeProvider({
                        id: data.id,
                        providerId: data.vcsProvider.id,
                        providerName: data.vcsProvider.name,
                        providerAutoCreateWebhooks: data.vcsProvider.autoCreateWebhooks,
                        ...savedFormData
                    })}
                >
                    Change provider
                </Button>
            </Box>
            {error && (
                <Alert severity={error.severity} sx={{ mb: 3 }}>
                    {error.message}
                </Alert>
            )}
            <VCSProviderLinkForm data={formData} onChange={setFormData} repositoryPathReadOnly />
            <Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: 1, mt: 3 }}>
                {dirty && <Button size="small" color="inherit" disabled={isInFlight} onClick={onDiscard}>Discard changes</Button>}
                <Button size="small" variant="outlined" color="primary" disabled={!dirty} loading={isInFlight} onClick={onSave}>Save changes</Button>
            </Box>
        </Paper>
    );
}

export default EditVCSProviderLink
