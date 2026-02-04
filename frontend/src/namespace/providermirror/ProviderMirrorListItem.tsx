import SmartToyIcon from '@mui/icons-material/SmartToy';
import { Box, Chip, ListItemButton, Tooltip, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from "react-relay/hooks";
import { Link } from 'react-router-dom';
import Gravatar from '../../common/Gravatar';
import Timestamp from '../../common/Timestamp';
import { ProviderMirrorListItemFragment_mirror$key } from './__generated__/ProviderMirrorListItemFragment_mirror.graphql';

interface Props {
    fragmentRef: ProviderMirrorListItemFragment_mirror$key
}

function ProviderMirrorListItem({ fragmentRef }: Props) {
    const theme = useTheme();

    const data = useFragment<ProviderMirrorListItemFragment_mirror$key>(graphql`
        fragment ProviderMirrorListItemFragment_mirror on TerraformProviderVersionMirror {
            id
            metadata {
                createdAt
            }
            version
            createdBy
            providerAddress
        }
    `, fragmentRef);

    return (
        <ListItemButton
            component={Link}
            to={data.id}
            sx={{
                borderBottom: `1px solid ${theme.palette.divider}`,
                borderLeft: `1px solid ${theme.palette.divider}`,
                borderRight: `1px solid ${theme.palette.divider}`,
                '&:last-child': { borderBottomLeftRadius: 4, borderBottomRightRadius: 4 }
            }}
        >
            <Box sx={{ flex: 1, [theme.breakpoints.down('lg')]: { paddingRight: 15 } }}>
                <Box display="flex" alignItems="center" gap={1}>
                    <Typography>{data.providerAddress}</Typography>
                    <Chip label={`v${data.version}`} size="small" />
                </Box>
                <Box display="flex" alignItems="center" mt={0.5}>
                    <Typography variant="caption" color="textSecondary">
                        Cached <Timestamp component="span" timestamp={data.metadata.createdAt} /> by
                    </Typography>
                    {data.createdBy.startsWith('trn:') ? (
                        <Tooltip title={data.createdBy}>
                            <SmartToyIcon sx={{ ml: 0.5, width: 16, height: 16, color: 'text.secondary' }} />
                        </Tooltip>
                    ) : (
                        <Tooltip title={data.createdBy}>
                            <Box><Gravatar width={16} height={16} sx={{ ml: 0.5 }} email={data.createdBy} /></Box>
                        </Tooltip>
                    )}
                </Box>
            </Box>
        </ListItemButton>
    );
}

export default ProviderMirrorListItem;
