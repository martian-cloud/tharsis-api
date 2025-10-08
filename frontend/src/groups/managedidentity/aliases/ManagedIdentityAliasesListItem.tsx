import { Box, ListItemButton, ListItemText, useTheme, Typography } from "@mui/material";
import graphql from 'babel-plugin-relay/macro';
import moment from 'moment';
import { useFragment } from "react-relay/hooks";
import { Link as RouterLink } from 'react-router-dom';
import ManagedIdentityTypeChip from "../ManagedIdentityTypeChip";
import { ManagedIdentityAliasesListItemFragment_managedIdentity$key } from "./__generated__/ManagedIdentityAliasesListItemFragment_managedIdentity.graphql";

interface Props {
    fragmentRef: ManagedIdentityAliasesListItemFragment_managedIdentity$key
}

function ManagedIdentityAliasesListItem({ fragmentRef }: Props) {
    const theme = useTheme();

    const data = useFragment<ManagedIdentityAliasesListItemFragment_managedIdentity$key>(graphql`
        fragment ManagedIdentityAliasesListItemFragment_managedIdentity on ManagedIdentity {
            metadata {
                updatedAt
            }
            id
            name
            description
            type
            groupPath
        }
    `, fragmentRef);

    return (
        <ListItemButton
            dense
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
            <ListItemText
                primary={<Box>
                    <Box display="flex">
                        <Typography sx={{ marginRight: 1 }}>{data.name}</Typography>
                        <ManagedIdentityTypeChip type={data.type} />
                    </Box>
                    <Box>
                        <Typography color="textSecondary">{data.description}</Typography>
                    </Box>
                </Box>}
                secondary={data.groupPath}
            />
            <Typography variant="body2" color="textSecondary">
                {moment(data.metadata.updatedAt as moment.MomentInput).fromNow()}
            </Typography>
        </ListItemButton>
    );
}

export default ManagedIdentityAliasesListItem
