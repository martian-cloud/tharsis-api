import { Box, Chip, ListItemIcon, Tooltip, Typography } from '@mui/material';
import Link from '@mui/material/Link';
import ListItem from '@mui/material/ListItem';
import { useTheme } from '@mui/material/styles';
import graphql from 'babel-plugin-relay/macro';
import { Terraform as TerraformIcon } from 'mdi-material-ui';
import { useFragment } from "react-relay/hooks";
import { Link as LinkRouter } from 'react-router-dom';
import Gravatar from '../common/Gravatar';
import Timestamp from '../common/Timestamp';
import { TerraformModuleSearchListItemFragment_module$key } from './__generated__/TerraformModuleSearchListItemFragment_module.graphql';

interface Props {
    fragmentRef: TerraformModuleSearchListItemFragment_module$key
}

function TerraformModuleSearchListItem(props: Props) {
    const theme = useTheme();

    const data = useFragment<TerraformModuleSearchListItemFragment_module$key>(graphql`
        fragment TerraformModuleSearchListItemFragment_module on TerraformModule {
            id
            name
            system
            registryNamespace
            private
            latestVersion {
                version
                createdBy
                metadata {
                    createdAt
                }
            }
        }
    `, props.fragmentRef);

    return (
        <ListItem
            button
            component={LinkRouter}
            to={`/module-registry/${data.registryNamespace}/${data.name}/${data.system}`}
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
                <TerraformIcon color="disabled" />
            </ListItemIcon>
            <Box flex={1} display="flex" justifyContent="space-between" alignItems="center">
                <Box>
                    <Link
                        component="div"
                        underline="hover"
                        variant="body1"
                        color="textPrimary"
                        sx={{ fontWeight: "500" }}
                    >
                        {data.registryNamespace}/{data.name}/{data.system}
                    </Link>
                    <Box>
                        {data.latestVersion && <Box display="flex" alignItems="center">
                            <Typography variant="body2" color="textSecondary">
                                {data.latestVersion.version} published <Timestamp component="span" timestamp={data.latestVersion.metadata.createdAt}/> by
                            </Typography>
                            <Tooltip title={data.latestVersion.createdBy}>
                                <Box>
                                    <Gravatar width={16} height={16} sx={{ marginLeft: 1, marginRight: 1 }} email={data.latestVersion.createdBy} />
                                </Box>
                            </Tooltip>
                        </Box>}
                        {!data.latestVersion && <Typography variant="body2" color="textSecondary">
                            0 versions
                        </Typography>}
                    </Box>
                </Box>
                {data.private && <Chip sx={{ marginLeft: 2 }} variant="outlined" color="warning" size="small" label="private" />}
            </Box>
        </ListItem>
    );
}

export default TerraformModuleSearchListItem;
