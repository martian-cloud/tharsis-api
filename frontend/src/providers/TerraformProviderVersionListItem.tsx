import { Box, Chip, ListItem, ListItemText, Tooltip, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from "react-relay/hooks";
import { Link as RouterLink } from 'react-router-dom';
import Gravatar from '../common/Gravatar';
import Timestamp from '../common/Timestamp';
import { TerraformProviderVersionListItemFragment_version$key } from './__generated__/TerraformProviderVersionListItemFragment_version.graphql';

interface Props {
    fragmentRef: TerraformProviderVersionListItemFragment_version$key
}

function TerraformProviderVersionListItem(props: Props) {
    const theme = useTheme();

    const data = useFragment<TerraformProviderVersionListItemFragment_version$key>(graphql`
        fragment TerraformProviderVersionListItemFragment_version on TerraformProviderVersion {
            metadata {
                createdAt
            }
            id
            version
            createdBy
            latest
            provider {
                name
                registryNamespace
            }
        }
    `, props.fragmentRef);

    return (
        <ListItem
            button
            component={RouterLink}
            to={`/provider-registry/${data.provider.registryNamespace}/${data.provider.name}/${data.version}`}
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

export default TerraformProviderVersionListItem;
