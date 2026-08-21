import { Box, Chip, ListItemButton, ListItemIcon, Tooltip, Typography } from '@mui/material';
import Link from '@mui/material/Link';
import { useTheme } from '@mui/material/styles';
import graphql from 'babel-plugin-relay/macro';
import { PackageVariantClosed as PackageIcon } from 'mdi-material-ui';
import { useFragment } from "react-relay/hooks";
import { Link as LinkRouter } from 'react-router-dom';
import Gravatar from '../common/Gravatar';
import Timestamp from '../common/Timestamp';
import PackageKindPill from './PackageKindPill';
import { PackageSearchListItemFragment_package$key } from './__generated__/PackageSearchListItemFragment_package.graphql';

interface Props {
    fragmentRef: PackageSearchListItemFragment_package$key
}

function PackageSearchListItem(props: Props) {
    const theme = useTheme();

    const data = useFragment<PackageSearchListItemFragment_package$key>(graphql`
        fragment PackageSearchListItemFragment_package on Package {
            id
            name
            groupPath
            kind
            visibility
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
        <ListItemButton
            component={LinkRouter}
            to={`/package-registry/${data.id}`}
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
                            {data.groupPath}/{data.name}
                        </Link>
                        {/* The pill trails the name rather than leading it as it does on a policy card,
                            because a list is scanned down the name column. */}
                        <PackageKindPill kind={data.kind} />
                    </Box>
                    <Box>
                        {data.latestVersion && <Typography component="div" variant="body2" color="textSecondary">
                            {data.latestVersion.version} published <Timestamp component="span" timestamp={data.latestVersion.metadata.createdAt} /> by
                            <Tooltip title={data.latestVersion.createdBy}>
                                <Box sx={{ display: 'inline-flex', verticalAlign: 'middle', mx: 1 }}>
                                    <Gravatar width={16} height={16} email={data.latestVersion.createdBy} />
                                </Box>
                            </Tooltip>
                        </Typography>}
                        {!data.latestVersion && <Typography variant="body2" color="textSecondary">
                            0 versions
                        </Typography>}
                    </Box>
                </Box>
                <Chip sx={{ marginLeft: 2 }} variant="outlined" size="small" label={data.visibility} />
            </Box>
        </ListItemButton>
    );
}

export default PackageSearchListItem;
