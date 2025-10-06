import { Avatar, Box, ListItem, ListItemText, Typography, useTheme } from '@mui/material';
import teal from '@mui/material/colors/teal';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from "react-relay/hooks";
import { Link as RouterLink } from 'react-router-dom';
import Timestamp from '../../common/Timestamp';
import { ServiceAccountListItemFragment_serviceAccount$key } from './__generated__/ServiceAccountListItemFragment_serviceAccount.graphql';

interface Props {
    fragmentRef: ServiceAccountListItemFragment_serviceAccount$key
    inherited: boolean
}

function ServiceAccountListItem({ fragmentRef, inherited }: Props) {
    const theme = useTheme();

    const data = useFragment<ServiceAccountListItemFragment_serviceAccount$key>(graphql`
        fragment ServiceAccountListItemFragment_serviceAccount on ServiceAccount {
            metadata {
                updatedAt
            }
            id
            name
            description
            resourcePath
            groupPath
        }
    `, fragmentRef);

    return (
        <ListItem
            button
            component={RouterLink}
            to={`/groups/${data.groupPath}/-/service_accounts/${data.id}`}
            sx={{
                borderBottom: `1px solid ${theme.palette.divider}`,
                borderLeft: `1px solid ${theme.palette.divider}`,
                borderRight: `1px solid ${theme.palette.divider}`,
                '&:last-child': {
                    borderBottomLeftRadius: 4,
                    borderBottomRightRadius: 4
                }
            }}>
            <Avatar variant="rounded" sx={{ width: 32, height: 32, bgcolor: teal[200], marginRight: 2 }}>
                {data.name[0].toUpperCase()}
            </Avatar>
            <ListItemText
                primary={<Box>
                    <Typography fontWeight={500}>{data.name}</Typography>
                    {data.description && <Typography variant="body2" color="textSecondary">{data.description}</Typography>}
                    {inherited && <Typography mt={0.5} color="textSecondary" variant="caption">Inherited from group <strong>{data.groupPath}</strong></Typography>}
                </Box>} />
            <Timestamp variant="body2" color="textSecondary" timestamp={data.metadata.updatedAt} />
        </ListItem>
    );
}

export default ServiceAccountListItem;
