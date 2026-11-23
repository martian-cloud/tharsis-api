import { Box, Link, ListItemButton, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { Link as RouterLink } from 'react-router-dom';
import NamespaceAvatar from '../../common/NamespaceAvatar';
import Timestamp from '../../common/Timestamp';
import { VCSProviderWorkspaceListItemFragment_workspace$key } from './__generated__/VCSProviderWorkspaceListItemFragment_workspace.graphql';

interface Props {
    fragmentRef: VCSProviderWorkspaceListItemFragment_workspace$key
    last?: boolean
}

function VCSProviderWorkspaceListItem({ fragmentRef, last }: Props) {
    const data = useFragment<VCSProviderWorkspaceListItemFragment_workspace$key>(graphql`
        fragment VCSProviderWorkspaceListItemFragment_workspace on Workspace {
            metadata {
                updatedAt
            }
            name
            description
            fullPath
        }
    `, fragmentRef);

    return (
        <ListItemButton divider={!last} component={RouterLink} to={`/groups/${data.fullPath}`}>
            <NamespaceAvatar name={data.name} type="workspace" sx={{ marginRight: 2 }} />
            <Box sx={{
                display: 'flex',
                flexGrow: 1,
                minWidth: 0,
                flexDirection: { xs: 'column', md: 'row' },
                alignItems: { md: 'center' },
                justifyContent: 'space-between',
                gap: { xs: 0.5, md: 2 }
            }}>
                <Box sx={{ minWidth: 0 }}>
                    <Link component="div" underline="hover" variant="body1" color="textPrimary" sx={{ wordBreak: 'break-word' }}>
                        {data.fullPath}
                    </Link>
                    <Typography variant="body2" color="textSecondary">{data.description}</Typography>
                </Box>
                <Timestamp variant="body2" color="textSecondary" timestamp={data.metadata.updatedAt} />
            </Box>
        </ListItemButton>
    );
}

export default VCSProviderWorkspaceListItem;
