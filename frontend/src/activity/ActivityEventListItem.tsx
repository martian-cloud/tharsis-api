import { Avatar, Box, ListItem, ListItemIcon, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from "react-relay/hooks";
import Gravatar from '../common/Gravatar';
import Timestamp from '../common/Timestamp';
import { ActivityEventListItemFragment_event$key } from './__generated__/ActivityEventListItemFragment_event.graphql';

interface Props {
    fragmentRef: ActivityEventListItemFragment_event$key
    icon: React.ReactNode
    primary: React.ReactNode
    secondary?: React.ReactNode
}

function ActivityEventListItem({ fragmentRef, icon, primary, secondary }: Props) {
    const theme = useTheme();

    const data = useFragment<ActivityEventListItemFragment_event$key>(graphql`
        fragment ActivityEventListItemFragment_event on ActivityEvent {
            metadata {
                createdAt
            }
            id
            initiator {
                __typename
                ...on User {
                    username
                    email
                }
                ...on ServiceAccount {
                    name
                    resourcePath
                }
            }
        }
    `, fragmentRef);

    return (
        <ListItem
            sx={{
                paddingTop: 2,
                paddingBottom: 2,
                borderBottom: `1px solid ${theme.palette.divider}`,
                '&:last-child': {
                    borderBottomWidth: 0
                }
            }}>
            <ListItemIcon sx={{ minWidth: 64 }}>
                <Avatar sx={{ backgroundColor: 'inherit', color: theme.palette.text.secondary, border: `1px ${theme.palette.divider} solid` }}>
                    {icon}
                </Avatar>
            </ListItemIcon>
            <Box>
                <Box>
                    <Box
                        sx={{
                            display: "flex",
                            alignItems: "center",
                            flex: 1,
                            mb: 0.5
                        }}>
                        {data.initiator.__typename === 'User' && <Gravatar width={18} height={18} email={data.initiator.email} />}
                        {data.initiator.__typename === 'ServiceAccount' && <Avatar
                            variant="rounded"
                            sx={{ width: 18, height: 18, bgcolor: 'avatar.default', fontSize: 14, fontWeight: 500 }}
                        >
                            {data.initiator.name[0].toUpperCase()}
                        </Avatar>}
                        <Typography
                            variant="body2"
                            component="span"
                            color="textSecondary"
                            sx={{
                                ml: 0.5,
                                fontWeight: 600
                            }}>
                            {data.initiator.__typename === 'User' && data.initiator.username}
                            {data.initiator.__typename === 'ServiceAccount' && data.initiator.resourcePath}
                        </Typography>
                        <Timestamp variant="body2" color="textSecondary" timestamp={data.metadata.createdAt} sx={{ ml: 2 }} />
                    </Box>
                    <Typography variant="body1" sx={{ wordBreak: 'break-word' }}>
                        {primary}
                    </Typography>
                </Box>
                {secondary && <Box
                    sx={{
                        mt: 1,
                        display: "flex"
                    }}>
                    {secondary}
                </Box>}
            </Box>
        </ListItem>
    );
}

export default ActivityEventListItem;
