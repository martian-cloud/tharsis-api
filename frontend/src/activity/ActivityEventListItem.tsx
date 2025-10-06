import { Avatar, Box, ListItem, ListItemIcon, Typography, useTheme } from '@mui/material';
import teal from '@mui/material/colors/teal';
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
                    <Box display="flex" alignItems="center" flex={1} mb={0.5}>
                        {data.initiator.__typename === 'User' && <Gravatar width={18} height={18} email={data.initiator.email} />}
                        {data.initiator.__typename === 'ServiceAccount' && <Avatar
                            variant="rounded"
                            sx={{ width: 18, height: 18, bgcolor: teal[200], fontSize: 14, fontWeight: 500 }}
                        >
                            {data.initiator.name[0].toUpperCase()}
                        </Avatar>}
                        <Typography ml={0.5} variant="body2" component="span" fontWeight={600} color="textSecondary">
                            {data.initiator.__typename === 'User' && data.initiator.username}
                            {data.initiator.__typename === 'ServiceAccount' && data.initiator.resourcePath}
                        </Typography>
                        <Timestamp ml={2} variant="body2" color="textSecondary" timestamp={data.metadata.createdAt} />
                    </Box>
                    <Typography variant="body1">{primary}</Typography>
                </Box>
                {secondary && <Box mt={1} display="flex">
                    {secondary}
                </Box>}
            </Box>
        </ListItem>
    );
}

export default ActivityEventListItem;
