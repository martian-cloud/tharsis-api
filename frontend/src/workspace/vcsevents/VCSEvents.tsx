import { Box, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { Suspense } from 'react';
import { useFragment } from 'react-relay/hooks';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import ListSkeleton from '../../skeletons/ListSkeleton';
import WorkspaceVCSWebhookAlert from '../WorkspaceVCSWebhookAlert';
import VCSEventList from './VCSEventList';
import { VCSEventsFragment_vcsEvents$key } from './__generated__/VCSEventsFragment_vcsEvents.graphql';

interface Props {
    fragmentRef: VCSEventsFragment_vcsEvents$key
}

function VCSEvents({ fragmentRef }: Props) {
    const data = useFragment<VCSEventsFragment_vcsEvents$key>(
        graphql`
        fragment VCSEventsFragment_vcsEvents on Workspace {
            id
            fullPath
            workspaceVcsProviderLink {
                id
                vcsProvider {
                    type
                }
            }
            ...WorkspaceVCSWebhookAlertFragment_workspace
        }
        `, fragmentRef);

    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={data.fullPath}
                childRoutes={[{ title: 'vcs events', path: 'vcs_events' }]}
            />
            <Typography variant="h5">VCS Events</Typography>
            <Typography color="textSecondary" sx={{ mb: 2.5 }}>
                Events received from the linked repository, including manually triggered ones.
            </Typography>
            <WorkspaceVCSWebhookAlert fragmentRef={data} />
            <Suspense fallback={<ListSkeleton rowCount={6} />}>
                <VCSEventList
                    workspaceId={data.id}
                    workspacePath={data.fullPath}
                    providerType={data.workspaceVcsProviderLink?.vcsProvider.type}
                />
            </Suspense>
        </Box>
    );
}

export default VCSEvents;
