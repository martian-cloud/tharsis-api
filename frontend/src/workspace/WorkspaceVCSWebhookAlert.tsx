import { Alert, AlertTitle, Box, Button } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { Link as RouterLink } from 'react-router-dom';
import { WorkspaceVCSWebhookAlertFragment_workspace$key } from './__generated__/WorkspaceVCSWebhookAlertFragment_workspace.graphql';

interface Props {
    fragmentRef: WorkspaceVCSWebhookAlertFragment_workspace$key
}

function WorkspaceVCSWebhookAlert({ fragmentRef }: Props) {
    const data = useFragment<WorkspaceVCSWebhookAlertFragment_workspace$key>(
        graphql`
        fragment WorkspaceVCSWebhookAlertFragment_workspace on Workspace {
            fullPath
            workspaceVcsProviderLink {
                usesLegacyWebhookToken
                webhookDisabled
            }
        }
        `, fragmentRef);

    if (!data.workspaceVcsProviderLink?.usesLegacyWebhookToken || data.workspaceVcsProviderLink.webhookDisabled) {
        return null;
    }

    return (
        <Alert
            sx={{ mb: 2 }}
            variant="outlined"
            severity='warning'>
            <AlertTitle>VCS webhook needs to be updated</AlertTitle>
            This workspace's VCS webhook is out of date. If it isn't updated, changes to your repository may stop starting runs.
            <Box sx={{ mt: 1, display: "flex", justifyContent: "flex-start" }}>
                <Button
                    variant="outlined"
                    color="warning"
                    size="small"
                    component={RouterLink}
                    to={`/groups/${data.fullPath}/-/settings?section=vcs-provider`}
                >
                    View VCS Settings
                </Button>
            </Box>
        </Alert>
    );
}

export default WorkspaceVCSWebhookAlert;
