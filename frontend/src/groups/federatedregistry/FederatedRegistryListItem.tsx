import { Box, ListItemAvatar, ListItemButton, ListItemText, Typography, useTheme } from '@mui/material';
import { FederatedRegistryIcon } from '../../common/Icons';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { Link as RouterLink } from 'react-router-dom';
import Timestamp from '../../common/Timestamp';
import { FederatedRegistryListItemFragment_federatedRegistry$key } from './__generated__/FederatedRegistryListItemFragment_federatedRegistry.graphql';

interface Props {
    fragmentRef: FederatedRegistryListItemFragment_federatedRegistry$key;
}

function FederatedRegistryListItem({ fragmentRef }: Props) {
    const theme = useTheme();

    const data = useFragment<FederatedRegistryListItemFragment_federatedRegistry$key>(graphql`
        fragment FederatedRegistryListItemFragment_federatedRegistry on FederatedRegistry {
            id
            hostname
            metadata {
                updatedAt
            }
            group {
                fullPath
            }
        }
    `, fragmentRef);

    const route = `/groups/${data.group.fullPath}/-/federated_registries/${data.id}`;

    return (
        <ListItemButton
            component={RouterLink}
            to={route}
            sx={{
                borderBottom: `1px solid ${theme.palette.divider}`,
                borderLeft: `1px solid ${theme.palette.divider}`,
                borderRight: `1px solid ${theme.palette.divider}`,
                '&:last-child': {
                    borderBottomLeftRadius: 4,
                    borderBottomRightRadius: 4
                }
            }}>
            <ListItemAvatar>
                <FederatedRegistryIcon />
            </ListItemAvatar>
            <ListItemText
                sx={{
                    flex: '1 1 auto',
                    overflow: 'hidden',
                    mr: 2
                }}
                primary={<Box>
                    <Typography
                        fontWeight={500}
                        noWrap={false}
                        sx={{
                            wordBreak: 'break-word',
                            maxWidth: '100%'
                        }}
                    >
                        {data.hostname}
                    </Typography>
                </Box>} />
            <Box sx={{ flexShrink: 0 }}>
                <Timestamp variant="body2" color="textSecondary" timestamp={data.metadata.updatedAt} />
            </Box>
        </ListItemButton>
    );
}

export default FederatedRegistryListItem;
