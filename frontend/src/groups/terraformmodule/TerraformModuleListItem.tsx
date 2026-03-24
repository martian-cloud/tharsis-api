import EditIcon from '@mui/icons-material/Edit';
import { Box, Chip, IconButton, ListItemButton, ListItemIcon, Typography } from '@mui/material';
import Link from '@mui/material/Link';
import { useTheme } from '@mui/material/styles';
import graphql from 'babel-plugin-relay/macro';
import { Terraform as TerraformIcon } from 'mdi-material-ui';
import { useFragment } from 'react-relay/hooks';
import { Link as RouterLink, useNavigate } from 'react-router-dom';
import LabelList from '../../workspace/labels/LabelList';
import { TerraformModuleListItemFragment_terraformModule$key } from './__generated__/TerraformModuleListItemFragment_terraformModule.graphql';

interface Props {
    fragmentRef: TerraformModuleListItemFragment_terraformModule$key
    inherited: boolean
}

function TerraformModuleListItem(props: Props) {
    const theme = useTheme();
    const navigate = useNavigate();

    const data = useFragment<TerraformModuleListItemFragment_terraformModule$key>(
        graphql`
        fragment TerraformModuleListItemFragment_terraformModule on TerraformModule
        {
            id
            name
            system
            registryNamespace
            private
            groupPath
            labels {
                key
                value
            }
            latestVersion {
                version
            }
        }
    `, props.fragmentRef);

    return (
        <>
            <ListItemButton
                component={RouterLink}
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
                            {data.name}/{data.system}
                        </Link>
                        <Box>
                            {data.latestVersion && <Box display="flex" alignItems="center">
                                <Typography variant="body2" color="textSecondary">
                                    {data.latestVersion.version}
                                </Typography>
                            </Box>}
                            {!data.latestVersion && <Typography variant="body2" color="textSecondary">
                                0 versions
                            </Typography>}
                        </Box>
                        {data.labels && data.labels.length > 0 && (
                            <Box mt={0.5}>
                                <LabelList
                                    labels={[...data.labels]}
                                    maxVisible={5}
                                    size="xs"
                                />
                            </Box>
                        )}
                        {props.inherited && <Typography mt={0.5} color="textSecondary" variant="caption">Inherited from group <strong>{data.groupPath}</strong></Typography>}
                    </Box>
                    <Box display="flex" alignItems="center" gap={1}>
                        <Chip sx={{ marginLeft: 2 }} variant="outlined" color={data.private ? 'warning' : 'default'} size="small" label={data.private ? 'private' : 'internal'} />
                        {!props.inherited && <IconButton
                            size="small"
                            onClick={(e: React.MouseEvent) => {
                                e.stopPropagation();
                                e.preventDefault();
                                navigate(`/groups/${data.groupPath}/-/terraform_modules/${data.name}/${data.system}/edit`);
                            }}
                        >
                            <EditIcon fontSize="small" />
                        </IconButton>}
                    </Box>
                </Box>
            </ListItemButton>
        </>
    );
}

export default TerraformModuleListItem;
