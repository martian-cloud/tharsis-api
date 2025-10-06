import { Box, Typography } from '@mui/material';
import NotificationButton from '../notifications/NotificationButton';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { GlobalNotificationPreferenceFragment_notificationPreference$key } from './__generated__/GlobalNotificationPreferenceFragment_notificationPreference.graphql';

interface Props {
    fragmentRef: GlobalNotificationPreferenceFragment_notificationPreference$key
}

function GlobalNotificationPreference({ fragmentRef }: Props) {

    const data = useFragment<GlobalNotificationPreferenceFragment_notificationPreference$key>(
        graphql`
            fragment GlobalNotificationPreferenceFragment_notificationPreference on GlobalUserPreferences {
                notificationPreference {
                    ...NotificationButtonFragment_notificationPreference
                }
            }
            `,
        fragmentRef
    );

    return (
        <Box>
            <Typography variant="subtitle1" gutterBottom>Global Notification</Typography>
            <Typography variant="body2" mb={2} color="textSecondary">
                By default, all groups and workspaces use the global notification preference
            </Typography>
            <NotificationButton
                path={null}
                isGlobalPreference
                fragmentRef={data.notificationPreference}
            />
        </Box>
    );
}

export default GlobalNotificationPreference;
