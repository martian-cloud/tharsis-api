import { useFragment, useLazyLoadQuery } from 'react-relay/hooks';
import graphql from 'babel-plugin-relay/macro';
import NotificationButton from './NotificationButton';
import { GroupNotificationPreferenceQuery } from './__generated__/GroupNotificationPreferenceQuery.graphql';
import { GroupNotificationPreferenceFragment_group$key } from './__generated__/GroupNotificationPreferenceFragment_group.graphql';

interface Props {
    fragmentRef: GroupNotificationPreferenceFragment_group$key;
}

function GroupNotificationPreference({ fragmentRef }: Props) {

    const group = useFragment<GroupNotificationPreferenceFragment_group$key>(
        graphql`
            fragment GroupNotificationPreferenceFragment_group on Group
            {
                fullPath
            }
        `,
        fragmentRef
    );

    const data = useLazyLoadQuery<GroupNotificationPreferenceQuery>(graphql`
        query GroupNotificationPreferenceQuery($groupPath: String!) {
            userPreferences {
                groupPreferences(first: 1, path: $groupPath) {
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
    `, { groupPath: group.fullPath }, { fetchPolicy: 'store-and-network' });

    const notificationPreference = data?.userPreferences?.groupPreferences?.edges?.[0]?.node?.notificationPreference as any;

    if (!notificationPreference) {
        console.error('GroupNotificationPreference: No data returned from useLazyLoadQuery, but data was expected');
        return null;
    }

    return (
        <NotificationButton
            path={group.fullPath}
            fragmentRef={notificationPreference}
        />
    );
}

export default GroupNotificationPreference;