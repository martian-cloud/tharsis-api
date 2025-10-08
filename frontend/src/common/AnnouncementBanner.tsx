import { Box, Stack } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useState, useMemo } from 'react';
import { useLazyLoadQuery } from 'react-relay/hooks';
import { useCookies } from 'react-cookie';
import { AnnouncementBannerQuery } from './__generated__/AnnouncementBannerQuery.graphql';
import AnnouncementAlert from './AnnouncementAlert';

// Cookie expiration for dismissed announcements (90 days)
const DISMISSED_ANNOUNCEMENTS_COOKIE_MAX_AGE = 60 * 60 * 24 * 90;

const query = graphql`
    query AnnouncementBannerQuery {
        announcements(active: true, sort: START_TIME_DESC, first: 5) {
            edges {
                node {
                    id
                    message
                    dismissible
                    type
                }
            }
        }
    }
`;

function AnnouncementBanner() {
    const [cookies, setCookie] = useCookies(['tharsis_dismissed_announcements']);

    const [dismissedAnnouncements, setDismissedAnnouncements] = useState<Set<string>>(() => {
        const dismissed = cookies.tharsis_dismissed_announcements;
        return dismissed && Array.isArray(dismissed) ? new Set(dismissed) : new Set();
    });

    const data = useLazyLoadQuery<AnnouncementBannerQuery>(
        query,
        {},
        { fetchPolicy: 'store-and-network' }
    );

    const activeAnnouncements = useMemo(() =>
        data.announcements.edges
            ?.flatMap(edge =>
                edge?.node && !dismissedAnnouncements.has(edge.node.id) ? [edge.node] : []
            ) || [],
        [data.announcements.edges, dismissedAnnouncements]
    );

    if (activeAnnouncements.length === 0) {
        return null;
    }

    const handleDismiss = (id: string) => {
        const newDismissed = new Set([...dismissedAnnouncements, id]);
        setDismissedAnnouncements(newDismissed);

        setCookie('tharsis_dismissed_announcements', Array.from(newDismissed), {
            path: '/',
            sameSite: 'strict',
            maxAge: DISMISSED_ANNOUNCEMENTS_COOKIE_MAX_AGE
        });
    };

    return (
        <Box sx={{ padding: 2 }}>
            <Stack spacing={1}>
                {activeAnnouncements.map(announcement => (
                    <AnnouncementAlert
                        key={announcement.id}
                        id={announcement.id}
                        message={announcement.message}
                        type={announcement.type}
                        dismissible={announcement.dismissible}
                        onDismiss={handleDismiss}
                    />
                ))}
            </Stack>
        </Box>
    );
}

export default AnnouncementBanner;
