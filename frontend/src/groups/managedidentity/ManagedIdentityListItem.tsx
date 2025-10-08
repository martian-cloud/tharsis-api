import { Box, Chip, ListItemButton, ListItemText, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from "react-relay/hooks";
import { Link as RouterLink } from 'react-router-dom';
import Timestamp from '../../common/Timestamp';
import ManagedIdentityTypeChip from './ManagedIdentityTypeChip';
import { ManagedIdentityListItemFragment_managedIdentity$key } from './__generated__/ManagedIdentityListItemFragment_managedIdentity.graphql';

interface Props {
    fragmentRef: ManagedIdentityListItemFragment_managedIdentity$key
    inherited: boolean
}

function ManagedIdentityListItem({ fragmentRef, inherited }: Props) {
    const theme = useTheme();

    const data = useFragment<ManagedIdentityListItemFragment_managedIdentity$key>(graphql`
        fragment ManagedIdentityListItemFragment_managedIdentity on ManagedIdentity {
            metadata {
                updatedAt
            }
            id
            isAlias
            name
            description
            type
            resourcePath
            groupPath
        }
    `, fragmentRef);

    return (
        <ListItemButton
            component={RouterLink}
            to={`/groups/${data.groupPath}/-/managed_identities/${data.id}`}
            sx={{
                borderBottom: `1px solid ${theme.palette.divider}`,
                borderLeft: `1px solid ${theme.palette.divider}`,
                borderRight: `1px solid ${theme.palette.divider}`,
                '&:last-child': {
                    borderBottomLeftRadius: 4,
                    borderBottomRightRadius: 4
                }
            }}>
            <Box minWidth={70}>
                <ManagedIdentityTypeChip mr={1} type={data.type} />
            </Box>
            <ListItemText
                primary={<Box>
                    <Box display="flex">
                        <Typography fontWeight={500}>{data.name}</Typography>
                        {data.isAlias && <Chip sx={{ ml: 1 }} label="alias" color="secondary" size="small" />}
                    </Box>
                    {data.description && <Typography variant="body2" color="textSecondary">{data.description}</Typography>}
                    {inherited && <Typography mt={0.5} color="textSecondary" variant="caption">Inherited from group <strong>{data.groupPath}</strong></Typography>}
                </Box>}
            />
            <Timestamp variant="body2" color="textSecondary" timestamp={data.metadata.updatedAt} />
        </ListItemButton>
    );
}

export default ManagedIdentityListItem
