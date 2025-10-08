import { useState } from 'react';
import { Alert, Box, Button, Collapse, Dialog, DialogActions, DialogTitle, DialogContent, IconButton, Typography } from '@mui/material'
import CopyIcon from '@mui/icons-material/ContentCopy';
import { useFragment } from 'react-relay';
import graphql from 'babel-plugin-relay/macro';
import { Link as RouterLink } from 'react-router-dom';
import SettingsToggleButton from '../../../common/SettingsToggleButton';
import EditVCSProviderLink from './EditVCSProviderLink';
import NewVCSProviderLink from './NewVCSProviderLink';
import { WorkspaceVCSProviderSettingsFragment_workspace$key } from './__generated__/WorkspaceVCSProviderSettingsFragment_workspace.graphql';

export interface WebhooksData {
    url: string
    token: string | null | undefined
    type: string | undefined
}

interface Props {
    fragmentRef: WorkspaceVCSProviderSettingsFragment_workspace$key
}

interface WebhooksDialogProps {
    open: boolean
    onClose: (confirm?: boolean) => void
    webhooksData: WebhooksData | null
}

function WebhooksDialog(props: WebhooksDialogProps){
    const { open, onClose, webhooksData } = props

    if (webhooksData) {
        return (
            <Dialog
                open={open}
                keepMounted
                maxWidth="sm"
            >
                <DialogTitle>Manually Configure Webhooks</DialogTitle>
                <DialogContent dividers>
                    <Box sx={{ mb: 3 }}>
                        <Typography variant="subtitle1">Copy this URL {webhooksData.type === 'gitlab' ? 'and token' : null} to manually configure webhooks in your VCS provider</Typography>
                    </Box>
                    <Box sx={{ mb: 3 }}>
                        <Box display="flex" flexDirection="row">
                            <Typography variant="subtitle1">Webhook URL</Typography>
                            <IconButton sx={{ padding: '4px' }} onClick={() => navigator.clipboard.writeText(webhooksData.url)}>
                                <CopyIcon sx={{ width: 16, height: 16 }} />
                            </IconButton>
                        </Box>
                        <Typography noWrap>{webhooksData?.url}</Typography>
                    </Box>
                    {webhooksData.type === 'gitlab' ? <Box sx={{ mb: 3 }}>
                        <Box display="flex" flexDirection="row">
                            <Typography variant="subtitle1">Webhook token</Typography>
                            <IconButton sx={{ padding: '4px' }} onClick={() => navigator.clipboard.writeText(webhooksData.token || '')}>
                                <CopyIcon sx={{ width: 16, height: 16 }} />
                            </IconButton>
                        </Box>
                        <Typography noWrap>{webhooksData.token}</Typography>
                    </Box> : null}
                    <Alert sx={{ marginBottom: 2 }} severity="warning">
                        Closing this dialog will cause the URL {webhooksData?.type === 'gitlab' ? 'and token' : null} to be lost. If you intend to manually configure your webhooks, ensure you have copied this information.
                    </Alert>
                </DialogContent>
                <DialogActions>
                    <Button size='small' variant='outlined' color="inherit" onClick={() => onClose()}>
                        Close
                    </Button>
                </DialogActions>
            </Dialog>
        )
    } else {
        return null
    }
}

function WorkspaceVCSProviderSettings({ fragmentRef }: Props) {
    const [webhookObj, setWebhookObj] = useState<WebhooksData | null>(null);
    const [openDialog, setOpenDialog] = useState<boolean>(false);
    const [showSettings, setShowSettings] = useState<boolean>(false);

    const data = useFragment<WorkspaceVCSProviderSettingsFragment_workspace$key>(
        graphql`
        fragment WorkspaceVCSProviderSettingsFragment_workspace on Workspace
        {
            workspaceVcsProviderLink {
                id
            }
            fullPath
            groupPath
            vcsProviders(first: 10, includeInherited: true) {
                edges {
                    node {
                        ... on VCSProvider {
                            id
                        }
                    }
                }
            }
            ...EditVCSProviderLinkFragment_workspace
            ...NewVCSProviderLinkFragment_workspace
        }
    `, fragmentRef
    )

    const handleWebhookDialog = (confirm: boolean, data: WebhooksData) => {
        setWebhookObj(data)
        setOpenDialog(confirm)
    }

    return (
        <Box>
            <SettingsToggleButton
                title="VCS Provider Link Settings"
                showSettings={showSettings}
                onToggle={() => setShowSettings(!showSettings)}
            />
            <Collapse
                in={showSettings}
                timeout="auto"
                unmountOnExit
            >
                <Box>
                    {data.workspaceVcsProviderLink && <EditVCSProviderLink fragmentRef={data} handleWebhookDialog={(confirm: boolean, data: WebhooksData) => handleWebhookDialog(confirm, data)} />}
                    {!data.workspaceVcsProviderLink && data.vcsProviders.edges && data.vcsProviders.edges.length > 0 && <NewVCSProviderLink fragmentRef={data}
                        handleWebhookDialog={(confirm: boolean, data: WebhooksData) => handleWebhookDialog(confirm, data)} />}
                    {!data.workspaceVcsProviderLink && data.vcsProviders.edges && data.vcsProviders.edges.length === 0 && <Box>
                        <Box sx={{ marginTop: 2, display: "flex", marginBottom: 2 }}>
                            <Box display="flex" flexDirection="column">
                                <Typography variant="subtitle1" gutterBottom>
                                    This workspace is not linked to a VCS Provider and there are no inherited VCS Providers. Get started by creating a VCS Provider in this group.
                                </Typography>
                                <Box marginTop={2}>
                                    <Button sx={{ minWidth: 200 }}
                                        component={RouterLink}
                                        variant="outlined"
                                        to={`../${data.groupPath}/-/vcs_providers/new`}
                                    >
                                        New VCS Provider
                                    </Button>
                                </Box>
                            </Box>
                        </Box>
                    </Box>}
                </Box>
            </Collapse>
            <WebhooksDialog
                webhooksData={webhookObj}
                open={openDialog}
                onClose={() => setOpenDialog(false)}
            />
        </Box>
    );
}

export default WorkspaceVCSProviderSettings;
