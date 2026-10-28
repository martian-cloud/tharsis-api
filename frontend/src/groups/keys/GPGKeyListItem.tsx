import DeleteIcon from '@mui/icons-material/Delete';
import { Box, IconButton, ListItem, ListItemIcon, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { KeyVariant as KeyIcon } from 'mdi-material-ui';
import { useFragment } from "react-relay/hooks";
import ActorDisplay from '../../common/ActorDisplay';
import Timestamp from '../../common/Timestamp';
import TRNButton from '../../common/TRNButton';
import { GPGKeyListItemFragment_key$key } from './__generated__/GPGKeyListItemFragment_key.graphql';

interface Props {
    fragmentRef: GPGKeyListItemFragment_key$key
    inherited: boolean
    onDelete: () => void
}

function GPGKeyListItem({ fragmentRef, inherited, onDelete }: Props) {
    const theme = useTheme();

    const data = useFragment<GPGKeyListItemFragment_key$key>(graphql`
        fragment GPGKeyListItemFragment_key on GPGKey {
            metadata {
                createdAt
                trn
            }
            id
            gpgKeyId
        	fingerprint
            createdByActor {
                ...ActorDisplayFragment_actor
            }
            groupPath
        }
    `, fragmentRef);

    return (
        <ListItem
            sx={{
                borderBottom: `1px solid ${theme.palette.divider}`,
                borderLeft: `1px solid ${theme.palette.divider}`,
                borderRight: `1px solid ${theme.palette.divider}`,
                '&:last-child': {
                    borderBottomLeftRadius: 4,
                    borderBottomRightRadius: 4
                }
            }}
        >
            <ListItemIcon sx={{ minWidth: 40 }}>
                <KeyIcon color="disabled" />
            </ListItemIcon>
            <Box sx={{ flex: 1, minWidth: 0 }}>
                <Typography sx={{ wordBreak: 'break-all' }}>
                    <Typography color="textSecondary" component="span">Key ID: </Typography>{data.gpgKeyId}
                </Typography>
                <Typography noWrap={false} sx={{
                    wordBreak: 'break-all',
                    [theme.breakpoints.down('md')]: {
                        display: 'none'
                    }
                }}>
                    <Typography color="textSecondary" component="span">Fingerprint: </Typography>{data.fingerprint}
                </Typography>
                <Box display="flex" alignItems="center" flexWrap="wrap" mt={1}>
                    <Typography variant="caption" color="textSecondary" sx={{ mr: 0.5 }}>
                        Added <Timestamp component="span" timestamp={data.metadata.createdAt} /> by
                    </Typography>
                    <ActorDisplay fragmentRef={data.createdByActor} variant="avatar" size={16} />
                </Box>
                {inherited && <Typography color="textSecondary" variant="caption">Inherited from group <strong>{data.groupPath}</strong></Typography>}
            </Box>
            <Box display="flex" alignItems="center" gap={1} sx={{ flexShrink: 0, ml: 1 }}>
                <TRNButton trn={data.metadata.trn} size="small" />
                <IconButton onClick={onDelete}>
                    <DeleteIcon />
                </IconButton>
            </Box>
        </ListItem >
    );
}

export default GPGKeyListItem;
