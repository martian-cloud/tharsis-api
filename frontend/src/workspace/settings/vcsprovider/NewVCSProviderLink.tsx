import { useState } from 'react';
import { Box, Typography } from '@mui/material'
import LoadingButton from '@mui/lab/LoadingButton';
import VCSProviderLinkForm from './VCSProviderLinkForm';
import graphql from 'babel-plugin-relay/macro';
import { VCSFormData } from './VCSProviderLinkForm';
import { MutationError } from '../../../common/error';
import { useSnackbar } from 'notistack';
import { WebhooksData } from './WorkspaceVCSProviderSettings';
import { GraphQLTaggedNode, useFragment, useMutation } from 'react-relay';
import { NewVCSProviderLinkMutation } from './__generated__/NewVCSProviderLinkMutation.graphql';
import { NewVCSProviderLinkFragment_workspace$key } from './__generated__/NewVCSProviderLinkFragment_workspace.graphql';

interface Props {
    fragmentRef: NewVCSProviderLinkFragment_workspace$key
    handleWebhookDialog: (confirm: boolean, data: WebhooksData) => void
}

export const NewMutation: GraphQLTaggedNode =
    graphql`
    mutation NewVCSProviderLinkMutation($input: CreateWorkspaceVCSProviderLinkInput!) {
        createWorkspaceVCSProviderLink(input: $input) {
            vcsProviderLink {
                workspace{
                    id
                    workspaceVcsProviderLink {
                        id
                        metadata {
                            createdAt
                        }
                        createdBy
                        repositoryPath
                        branch
                        moduleDirectory
                        tagRegex
                        globPatterns
                        autoSpeculativePlan
                        webhookDisabled
                    }
                }
                vcsProvider {
                    type
                    autoCreateWebhooks
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


function NewVCSProviderLink({ fragmentRef, handleWebhookDialog }: Props) {
    const { enqueueSnackbar } = useSnackbar();

    const [error, setError] = useState<MutationError>()
    const [formData, setFormData] = useState<VCSFormData>({
        id: '',
        repositoryPath: '',
        moduleDirectory: '',
        branch: '',
        tagRegex: '',
        globPatterns: [],
        autoSpeculativePlan: false,
        webhookDisabled: false,
        label: '',
        description: '',
        type: undefined
    })

    const workspace = useFragment<NewVCSProviderLinkFragment_workspace$key>(
        graphql`
        fragment NewVCSProviderLinkFragment_workspace on Workspace
        {
            fullPath
            ...VCSProviderLinkFormFragment_workspace
        }
        `, fragmentRef
    )

    const [newCommit, newIsInFlight] = useMutation<NewVCSProviderLinkMutation>(NewMutation)

    const onSave = () => {
        newCommit({
            variables: {
                input: {
                    repositoryPath: formData.repositoryPath,
                    workspacePath: workspace.fullPath,
                    moduleDirectory: formData.moduleDirectory,
                    providerId: formData.id,
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
    };

    return (
        <Box>
            <Typography variant="subtitle1" gutterBottom>To link this workspace to a VCS provider, get started by selecting a provider below.</Typography>
            <VCSProviderLinkForm
                error={error}
                data={formData}
                fragmentRef={workspace}
                onChange={(data: VCSFormData) => setFormData(data)}
            />
            <Box sx={{ mt: 2 }}>
                <LoadingButton
                    sx={{ mr: 2 }}
                    size="small"
                    disabled={!formData.id || formData.repositoryPath === ''}
                    loading={newIsInFlight}
                    variant="outlined"
                    color="primary"
                    onClick={onSave}>Save changes
                </LoadingButton>
            </Box>
        </Box>
    )
}

export default NewVCSProviderLink
