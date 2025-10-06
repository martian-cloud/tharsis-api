import DeleteIcon from '@mui/icons-material/Delete';
import { Box, IconButton, ListItem, ListItemIcon, Tooltip, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { KeyVariant as KeyIcon } from 'mdi-material-ui';
import { useFragment } from "react-relay/hooks";
import Gravatar from '../../common/Gravatar';
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
            createdBy
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
            secondaryAction={<Box display="flex" alignItems="center" gap={1}>
                <TRNButton trn={data.metadata.trn} size="small"/>
                <IconButton onClick={onDelete}>
                    <DeleteIcon />
                </IconButton>
            </Box>}
        >
            <ListItemIcon sx={{ minWidth: 40 }}>
                <KeyIcon color="disabled" />
            </ListItemIcon>
            <Box sx={{
                flex: 1,
                display: 'flex',
                flexDirection: 'row',
                [theme.breakpoints.down('lg')]: {
                    paddingRight: 15
                }
            }}>
                <Box>
                    <Typography>
                        <Typography color="textSecondary" component="span">Key ID: </Typography>{data.gpgKeyId}
                    </Typography>
                    <Typography noWrap={false} sx={{
                        [theme.breakpoints.down('md')]: {
                            display: 'none'
                        }
                    }}>
                        <Typography color="textSecondary" component="span">Fingerprint: </Typography>{data.fingerprint}
                    </Typography>
                    <Box display="flex" alignItems="center" mt={1}>
                        <Typography variant="caption" color="textSecondary">
                            Added <Timestamp component="span" timestamp={data.metadata.createdAt} /> by
                        </Typography>
                        <Tooltip title={data.createdBy}>
                            <Box>
                                <Gravatar width={16} height={16} sx={{ marginLeft: 1, marginRight: 1 }} email={data.createdBy} />
                            </Box>
                        </Tooltip>
                    </Box>
                    {inherited && <Typography color="textSecondary" variant="caption">Inherited from group <strong>{data.groupPath}</strong></Typography>}
                </Box>
            </Box>
        </ListItem >
    );
}

export default GPGKeyListItem;
