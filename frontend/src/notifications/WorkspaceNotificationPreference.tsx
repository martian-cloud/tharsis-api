import { useFragment, useLazyLoadQuery } from 'react-relay/hooks';
import graphql from 'babel-plugin-relay/macro';
import NotificationButton from './NotificationButton';
import { WorkspaceNotificationPreferenceQuery } from './__generated__/WorkspaceNotificationPreferenceQuery.graphql';
import { WorkspaceNotificationPreferenceFragment_workspace$key } from './__generated__/WorkspaceNotificationPreferenceFragment_workspace.graphql';

interface Props {
    fragmentRef: WorkspaceNotificationPreferenceFragment_workspace$key;
}

function WorkspaceNotificationPreference({ fragmentRef }: Props) {

    const workspace = useFragment<WorkspaceNotificationPreferenceFragment_workspace$key>(
        graphql`
            fragment WorkspaceNotificationPreferenceFragment_workspace on Workspace
            {
                fullPath
            }
        `,
        fragmentRef
    );

    const data = useLazyLoadQuery<WorkspaceNotificationPreferenceQuery>(graphql`
        query WorkspaceNotificationPreferenceQuery($workspacePath: String!) {
            userPreferences {
                workspacePreferences(first: 1, path: $workspacePath) {
                    edges {
                        node {
                            notificationPreference {
                                ...NotificationButtonFragment_notificationPreference
                            }
                        }
                    }
                }
            }
        }
    `, { workspacePath: workspace.fullPath }, { fetchPolicy: 'store-and-network' });

    const notificationPreference = data?.userPreferences?.workspacePreferences?.edges?.[0]?.node?.notificationPreference as any;

    if (!notificationPreference) {
        console.error('WorkspaceNotificationPreference: No data returned from useLazyLoadQuery, but data was expected');
        return null;
    }

    return (
        <NotificationButton
            path={workspace.fullPath}
            fragmentRef={notificationPreference}
        />
    );
}

export default WorkspaceNotificationPreference;