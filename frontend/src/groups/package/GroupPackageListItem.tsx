import DeleteIcon from '@mui/icons-material/Delete';
import EditIcon from '@mui/icons-material/Edit';
import { Box, Chip, IconButton, ListItemButton, ListItemIcon, Tooltip, Typography } from '@mui/material';
import Link from '@mui/material/Link';
import { useTheme } from '@mui/material/styles';
import graphql from 'babel-plugin-relay/macro';
import { PackageVariantClosed as PackageIcon } from 'mdi-material-ui';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { Link as RouterLink, useNavigate } from 'react-router-dom';
import PackageKindPill from '../../packages/PackageKindPill';
import { GroupPackageListItemFragment_package$key } from './__generated__/GroupPackageListItemFragment_package.graphql';

interface Props {
    fragmentRef: GroupPackageListItemFragment_package$key
    inherited: boolean
    // onDelete asks the list to confirm and delete this package. Like the edit action it is only
    // offered for packages this group owns.
    onDelete?: (pkg: { id: string, name: string }) => void
}

function GroupPackageListItem(props: Props) {
    const theme = useTheme();
    const navigate = useNavigate();

    const data = useFragment<GroupPackageListItemFragment_package$key>(
        graphql`
        fragment GroupPackageListItemFragment_package on Package
        {
            id
            name
            description
            kind
            visibility
            groupPath
            latestVersion {
                version
            }
        }
    `, props.fragmentRef);

    return (
        <ListItemButton
            component={RouterLink}
            to={`/groups/${data.groupPath}/-/packages/${data.id}`}
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
                <PackageIcon color="disabled" />
            </ListItemIcon>
            <Box flex={1} display="flex" justifyContent="space-between" alignItems="center">
                <Box sx={{ minWidth: 0 }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap' }}>
                        <Link
                            component="div"
                            underline="hover"
                            variant="body1"
                            color="textPrimary"
                            sx={{ fontWeight: "500" }}
                        >
                            {data.name}
                        </Link>
                        {/* The pill trails the name rather than leading it as it does on a policy card,
                            because a list is scanned down the name column. */}
                        <PackageKindPill kind={data.kind} />
                    </Box>
                    {data.description && <Typography variant="body2" color="textSecondary">{data.description}</Typography>}
                    {props.inherited && <Typography mt={0.5} color="textSecondary" variant="caption">Inherited from group <strong>{data.groupPath}</strong></Typography>}
                </Box>
                <Box display="flex" alignItems="center" gap={1}>
                    <Chip sx={{ marginLeft: 2 }} variant="outlined" size="small" label={data.visibility} />
                    {/* The row is a link, so the actions have to swallow the click that reaches them. */}
                    {!props.inherited && <IconButton
                        size="small"
                        onClick={(e: React.MouseEvent) => {
                            e.stopPropagation();
                            e.preventDefault();
                            navigate(`/groups/${data.groupPath}/-/packages/${data.id}/edit`);
                        }}
                    >
                        <EditIcon fontSize="small" />
                    </IconButton>}
                    {!props.inherited && props.onDelete && <Tooltip title="Delete package">
                        <IconButton
                            size="small"
                            onClick={(e: React.MouseEvent) => {
                                e.stopPropagation();
                                e.preventDefault();
                                props.onDelete?.({ id: data.id, name: data.name });
                            }}
                        >
                            <DeleteIcon fontSize="small" />
                        </IconButton>
                    </Tooltip>}
                </Box>
            </Box>
        </ListItemButton>
    );
}

export default GroupPackageListItem;
