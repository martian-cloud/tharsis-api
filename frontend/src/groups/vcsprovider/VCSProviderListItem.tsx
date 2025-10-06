import { Avatar, Box, ListItem, ListItemText, Typography, useTheme } from '@mui/material';
import teal from '@mui/material/colors/teal';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay';
import { Link as RouterLink } from 'react-router-dom';
import Timestamp from '../../common/Timestamp';
import { VCSProviderListItemFragment_vcsProvider$key } from './__generated__/VCSProviderListItemFragment_vcsProvider.graphql';

interface Props {
    fragmentRef: VCSProviderListItemFragment_vcsProvider$key
    inherited: boolean
}

function VCSProviderListItem({ fragmentRef, inherited }: Props) {
    const theme = useTheme();

    const data = useFragment<VCSProviderListItemFragment_vcsProvider$key>(
        graphql`
        fragment VCSProviderListItemFragment_vcsProvider on VCSProvider {
            metadata {
                updatedAt
            }
            id
            name
            description
            groupPath
        }
    `, fragmentRef)

    return (
        <ListItem
            button
            component={RouterLink}
            to={`/groups/${data.groupPath}/-/vcs_providers/${data.id}`}
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
                </Box>}
            />
            <Timestamp variant="body2" color="textSecondary" timestamp={data.metadata.updatedAt} />
        </ListItem>
    )
}

export default VCSProviderListItem
