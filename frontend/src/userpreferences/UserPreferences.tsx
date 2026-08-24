import { Typography } from '@mui/material';
import { PageLayoutProvider } from '../layout/PageLayoutContext';
import graphql from 'babel-plugin-relay/macro';
import { useFragment, PreloadedQuery, usePreloadedQuery } from 'react-relay/hooks';
import GlobalNotificationPreference from './GlobalNotificationPreference';
import UserSessions from './UserSessions';
import PreferenceSection from './PreferenceSection';
import { UserPreferencesQuery } from './__generated__/UserPreferencesQuery.graphql';
import { UserPreferencesFragment_preferences$key } from './__generated__/UserPreferencesFragment_preferences.graphql';

const query = graphql`
    query UserPreferencesQuery($first: Int, $after: String) {
        ...UserPreferencesFragment_preferences
    }
`;

interface Props {
    queryRef: PreloadedQuery<UserPreferencesQuery>;
}

function UserPreferences({ queryRef }: Props) {
    const queryData = usePreloadedQuery<UserPreferencesQuery>(query, queryRef);

    const data = useFragment<UserPreferencesFragment_preferences$key>(
        graphql`
            fragment UserPreferencesFragment_preferences on Query {
                userPreferences {
                    globalPreferences {
                        ...GlobalNotificationPreferenceFragment_notificationPreference
                    }
                }
                me {
                    ... on User {
                        ...UserSessionsFragment_user
                    }
                }
            }
        `,
        queryData
    );

    return (
        <PageLayoutProvider>
            <Typography marginBottom={4} variant="h5" gutterBottom>
                Preferences
            </Typography>
            
            <PreferenceSection title="Notifications">
                <GlobalNotificationPreference fragmentRef={data.userPreferences.globalPreferences} />
            </PreferenceSection>

            {data.me && (
                <PreferenceSection title="Security" isLast>
                    <UserSessions fragmentRef={data.me} />
                </PreferenceSection>
            )}
        </PageLayoutProvider>
    );
}

export default UserPreferences;
