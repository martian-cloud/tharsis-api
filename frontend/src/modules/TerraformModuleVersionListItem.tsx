import { Box, Chip, ListItem, ListItemText, Tooltip, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from "react-relay/hooks";
import { Link as RouterLink } from 'react-router-dom';
import Gravatar from '../common/Gravatar';
import Timestamp from '../common/Timestamp';
import { TerraformModuleVersionListItemFragment_version$key } from './__generated__/TerraformModuleVersionListItemFragment_version.graphql';

interface Props {
    fragmentRef: TerraformModuleVersionListItemFragment_version$key
}

function TerraformModuleVersionListItem(props: Props) {
    const theme = useTheme();

    const data = useFragment<TerraformModuleVersionListItemFragment_version$key>(graphql`
        fragment TerraformModuleVersionListItemFragment_version on TerraformModuleVersion {
            metadata {
                createdAt
            }
            id
            version
            createdBy
            latest
            module {
                name
                registryNamespace
                system
            }
        }
    `, props.fragmentRef);

    return (
        <ListItem
            button
            component={RouterLink}
            to={`/module-registry/${data.module.registryNamespace}/${data.module.name}/${data.module.system}/${data.version}`}
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
                primary={<Box display="flex" alignItems="center">
                    <Typography>{data.version}</Typography>
                    {data.latest && <Chip size="small" color="secondary" sx={{ marginLeft: 1 }} label="latest" />}
                </Box>} />
            <Box display="flex" alignItems="center">
                <Typography variant="body2" color="textSecondary" sx={{ marginRight: 1 }}>
                    <Timestamp component="span" timestamp={data.metadata.createdAt} /> by
                </Typography>
                <Tooltip title={data.createdBy}>
                    <Box>
                        <Gravatar width={20} height={20} email={data.createdBy} />
                    </Box>
                </Tooltip>
            </Box>
        </ListItem>
    )
}

export default TerraformModuleVersionListItem;
