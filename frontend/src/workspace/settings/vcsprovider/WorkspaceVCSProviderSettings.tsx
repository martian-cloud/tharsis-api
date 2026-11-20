import { Box, Button, Paper, Stack, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useState } from 'react';
import { useFragment } from 'react-relay';
import { Link as RouterLink } from 'react-router-dom';
import SettingsSection from '../../../common/SettingsSection';
import ChangeVCSProviderDialog, { ChangeVCSProviderLink } from './ChangeVCSProviderDialog';
import EditVCSProviderLink from './EditVCSProviderLink';
import ManualWebhookDialog, { ManualWebhookDetails } from './ManualWebhookDialog';
import NewVCSProviderLink from './NewVCSProviderLink';
import VCSProviderLinkDangerZone from './VCSProviderLinkDangerZone';
import VCSProviderLinkWebhook from './VCSProviderLinkWebhook';
import { WorkspaceVCSProviderSettingsFragment_workspace$key } from './__generated__/WorkspaceVCSProviderSettingsFragment_workspace.graphql';

interface Props {
    fragmentRef: WorkspaceVCSProviderSettingsFragment_workspace$key
}

function WorkspaceVCSProviderSettings({ fragmentRef }: Props) {
    const [linking, setLinking] = useState(false);
    const [webhookDetails, setWebhookDetails] = useState<ManualWebhookDetails | null>(null);
    const [changeProviderLink, setChangeProviderLink] = useState<ChangeVCSProviderLink | null>(null);

    const data = useFragment<WorkspaceVCSProviderSettingsFragment_workspace$key>(
        graphql`
        fragment WorkspaceVCSProviderSettingsFragment_workspace on Workspace
        {
            fullPath
            groupPath
            workspaceVcsProviderLink {
                id
                ...EditVCSProviderLinkFragment_link
                ...VCSProviderLinkWebhookFragment_link
                ...VCSProviderLinkDangerZoneFragment_link
            }
            vcsProviders(first: 1, includeInherited: true) {
                edges {
                    node {
                        ... on VCSProvider {
                            id
                        }
                    }
                }
            }
        }
    `, fragmentRef
    );

    const link = data.workspaceVcsProviderLink;
    const hasProviders = (data.vcsProviders?.edges?.length ?? 0) > 0;

    return (
        <Box>
            <SettingsSection section="vcs-provider" title="VCS Provider Link Settings">
                {link && (
                    <Stack spacing={3} sx={{ mt: 2 }}>
                        <EditVCSProviderLink key={link.id} fragmentRef={link} onChangeProvider={setChangeProviderLink} />
                        <VCSProviderLinkWebhook fragmentRef={link} onShowWebhookDetails={setWebhookDetails} />
                        <VCSProviderLinkDangerZone fragmentRef={link} />
                    </Stack>
                )}
                {!link && linking && (
                    <NewVCSProviderLink
                        workspacePath={data.fullPath}
                        onLinked={() => setLinking(false)}
                        onCancel={() => setLinking(false)}
                        onShowWebhookDetails={setWebhookDetails}
                    />
                )}
                {!link && !linking && (
                    <Paper sx={{ p: 4, mt: 2, textAlign: 'center' }}>
                        <Typography variant="h6" gutterBottom>No repository linked</Typography>
                        {hasProviders ? <>
                            <Typography color="textSecondary" sx={{ mb: 3 }}>
                                Link a GitHub or GitLab repository to start runs from pushes, tags and merge requests.
                            </Typography>
                            <Button variant="contained" color="secondary" onClick={() => setLinking(true)}>Link a repository</Button>
                        </> : <>
                            <Typography color="textSecondary" sx={{ mb: 3 }}>
                                There are no VCS providers in this group or its parent groups. Create one to link a repository.
                            </Typography>
                            <Button
                                variant="outlined"
                                color="secondary"
                                component={RouterLink}
                                to={`../${data.groupPath}/-/vcs_providers/new`}
                            >
                                New VCS provider
                            </Button>
                        </>}
                    </Paper>
                )}
            </SettingsSection>
            {changeProviderLink && (
                <ChangeVCSProviderDialog
                    link={changeProviderLink}
                    workspacePath={data.fullPath}
                    onClose={() => setChangeProviderLink(null)}
                    onShowWebhookDetails={setWebhookDetails}
                />
            )}
            {webhookDetails && <ManualWebhookDialog details={webhookDetails} onClose={() => setWebhookDetails(null)} />}
        </Box>
    );
}

export default WorkspaceVCSProviderSettings;
