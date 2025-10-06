import LoadingButton from '@mui/lab/LoadingButton';
import { Box, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import { useState } from 'react';
import { useFragment, useMutation } from 'react-relay';
import { MutationError } from '../../../common/error';
import { NewMutation } from './NewVCSProviderLink';
import VCSProviderLinkForm, { VCSFormData } from './VCSProviderLinkForm';
import { StyledCode } from '../../../common/StyledCode';
import { WebhooksData } from './WorkspaceVCSProviderSettings';
import { NewVCSProviderLinkMutation } from './__generated__/NewVCSProviderLinkMutation.graphql';
import { EditVCSProviderLinkDeleteMutation } from './__generated__/EditVCSProviderLinkDeleteMutation.graphql';
import { EditVCSProviderLinkFragment_workspace$key } from './__generated__/EditVCSProviderLinkFragment_workspace.graphql';
import { EditVCSProviderLinkMutation } from './__generated__/EditVCSProviderLinkMutation.graphql';

interface Props {
    fragmentRef: EditVCSProviderLinkFragment_workspace$key
    handleWebhookDialog: (confirm: boolean, data: WebhooksData) => void
}

function EditVCSProviderLink({ fragmentRef, handleWebhookDialog }: Props) {
    const { enqueueSnackbar } = useSnackbar();

    const data = useFragment<EditVCSProviderLinkFragment_workspace$key>(
        graphql`
        fragment EditVCSProviderLinkFragment_workspace on Workspace
        {
            fullPath
            workspaceVcsProviderLink {
                id
                metadata {
                    createdAt
                }
                createdBy
                repositoryPath
                autoSpeculativePlan
                webhookDisabled
                moduleDirectory
                branch
                tagRegex
                globPatterns
                vcsProvider {
                    id
                    name
                    description
                    type
                    autoCreateWebhooks
                }
            }
            ...VCSProviderLinkFormFragment_workspace
        }
        `, fragmentRef
    )

    const [updateCommit, updateIsInFlight] = useMutation<EditVCSProviderLinkMutation>(
        graphql`
        mutation EditVCSProviderLinkMutation($input: UpdateWorkspaceVCSProviderLinkInput!) {
            updateWorkspaceVCSProviderLink(input: $input){
                vcsProviderLink{
                    id
                    repositoryPath
                    moduleDirectory
                    branch
                    tagRegex
                    globPatterns
                    autoSpeculativePlan
                    webhookDisabled
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `)
    const [error, setError] = useState<MutationError>()
    const [formData, setFormData] = useState<VCSFormData>({
        id: data.workspaceVcsProviderLink?.vcsProvider.id || '',
        repositoryPath: data.workspaceVcsProviderLink?.repositoryPath || '',
        moduleDirectory: data.workspaceVcsProviderLink?.moduleDirectory || '',
        branch: data.workspaceVcsProviderLink?.branch || '',
        tagRegex: data.workspaceVcsProviderLink?.tagRegex || '',
        globPatterns: data.workspaceVcsProviderLink?.globPatterns || [''],
        autoSpeculativePlan: data.workspaceVcsProviderLink?.autoSpeculativePlan || false,
        webhookDisabled: data.workspaceVcsProviderLink?.webhookDisabled || false,
        label: data.workspaceVcsProviderLink?.vcsProvider.name || '',
        description: data.workspaceVcsProviderLink?.vcsProvider.description || '',
        type: data.workspaceVcsProviderLink?.vcsProvider.type || ''
    })

    const onUpdate = () => {
        if (data.workspaceVcsProviderLink) {
            updateCommit({
                variables: {
                    input: {
                        id: data.workspaceVcsProviderLink.id,
                        moduleDirectory: formData.moduleDirectory,
                        branch: formData.branch,
                        tagRegex: formData.tagRegex,
                        globPatterns: formData.globPatterns,
                        autoSpeculativePlan: formData.autoSpeculativePlan,
                        webhookDisabled: formData.webhookDisabled
                    }
                },
                onCompleted: data => {
                    if (data.updateWorkspaceVCSProviderLink.problems.length) {
                        setError({
                            severity: 'warning',
                            message: data.updateWorkspaceVCSProviderLink.problems.map((problem: any) => problem.message).join('; ')
                        });
                    } else if (!data.updateWorkspaceVCSProviderLink.vcsProviderLink) {
                        setError({
                            severity: 'error',
                            message: "Unexpected error occurred"
                        });
                    } else {
                        enqueueSnackbar('VCS Provider Settings updated', { variant: 'success' });
                    }
                },
                onError: error => {
                    setError({
                        severity: 'error',
                        message: `Unexpected error occurred: ${error.message}`
                    });
                }
            });
        }
    };

    const [newCommit] = useMutation<NewVCSProviderLinkMutation>(NewMutation)

    const [deleteCommit, deleteIsInFlight] = useMutation<EditVCSProviderLinkDeleteMutation>(
        graphql`
        mutation EditVCSProviderLinkDeleteMutation($input: DeleteWorkspaceVCSProviderLinkInput!) {
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

    const onDeleteAndCreate = () => {

        const workspaceVcsProvider = data.workspaceVcsProviderLink as any

        deleteCommit({
            variables: {
                input: {
                    id: workspaceVcsProvider.id
                },
            },
            onCompleted: dataDelete => {
                if (dataDelete.deleteWorkspaceVCSProviderLink.problems.length) {
                    enqueueSnackbar(dataDelete.deleteWorkspaceVCSProviderLink.problems.map((problem: any) => problem.message).join('; '), { variant: 'warning' });
                } else {
                    if (formData.id !== '') {
                        newCommit({
                            variables: {
                                input: {
                                    repositoryPath: formData.repositoryPath,
                                    workspacePath: workspaceVcsProvider.workspace.fullPath,
                                    moduleDirectory: formData.moduleDirectory,
                                    providerId: formData.id || '',
                                    branch: formData.branch,
                                    tagRegex: formData.tagRegex,
                                    globPatterns: formData.globPatterns,
                                    autoSpeculativePlan: formData.autoSpeculativePlan,
                                    webhookDisabled: formData.webhookDisabled
                                },
                            },
                            onCompleted: data => {
                                if (data.createWorkspaceVCSProviderLink.problems.length) {
                                    setError({
                                        severity: 'warning',
                                        message: data.createWorkspaceVCSProviderLink.problems.map((problem: any) => problem.message).join('; ')
                                    });
                                } else if (!data.createWorkspaceVCSProviderLink.vcsProviderLink) {
                                    setError({
                                        severity: 'error',
                                        message: "Unexpected error occurred"
                                    });
                                } else {
                                    enqueueSnackbar(`Workspace VCS Provider Link created`, { variant: 'success' })
                                    if (!data.createWorkspaceVCSProviderLink.vcsProviderLink.vcsProvider.autoCreateWebhooks && data.createWorkspaceVCSProviderLink.webhookUrl) {
                                        handleWebhookDialog(true, { ...data, url: data.createWorkspaceVCSProviderLink.webhookUrl, token: data.createWorkspaceVCSProviderLink.webhookToken, type: data.createWorkspaceVCSProviderLink.vcsProviderLink.vcsProvider.type })
                                    }
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
                        enqueueSnackbar(`Workspace VCS Provider Link deleted`, { variant: 'success' })
                    }
                }
            },
            onError: error => {
                enqueueSnackbar(`Unexpected error occurred: ${error.message}`, { variant: 'error' });
            },
        });
    }

    const handleChanges = () => {
        if (data.workspaceVcsProviderLink?.vcsProvider.id === formData.id) {
            onUpdate()
        }
        else {
            onDeleteAndCreate()
        }
    }

    return (
        <Box>
            <Box>
                <Typography variant="subtitle1" gutterBottom>To link to different provider, select a provider from the drop-down menu, fill out the form, and click <StyledCode>Save Changes</StyledCode>.
                </Typography>
                <Typography variant="subtitle1" gutterBottom>To <strong>delete this link</strong>, empty the drop-down menu, and then click <StyledCode>Save Changes</StyledCode>.
                </Typography>
            </Box>
            <VCSProviderLinkForm
                viewMode
                error={error}
                data={formData}
                onChange={(data: VCSFormData) => setFormData(data)}
                fragmentRef={data}
            />
            <Box sx={{ mt: 2 }} display="flex" alignItems="baseline">
                <LoadingButton
                    size="small"
                    loading={updateIsInFlight || deleteIsInFlight}
                    variant="outlined"
                    color="primary"
                    onClick={handleChanges}
                >
                    Save changes
                </LoadingButton>
            </Box>
        </Box>
    )
}

export default EditVCSProviderLink
