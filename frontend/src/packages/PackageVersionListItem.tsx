import { Box, Chip, ListItemButton, ListItemText, Tooltip, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from "react-relay/hooks";
import Gravatar from '../common/Gravatar';
import Timestamp from '../common/Timestamp';
import { PackageVersionListItemFragment_version$key } from './__generated__/PackageVersionListItemFragment_version.graphql';

interface Props {
    fragmentRef: PackageVersionListItemFragment_version$key
    onSelect?: (versionId: string) => void
}

// A row's only job is choosing which version to look at. Editing and deleting act on the version being
// viewed and live in the page header, so they aren't repeated per row.
function PackageVersionListItem(props: Props) {
    const { onSelect } = props;
    const theme = useTheme();

    const data = useFragment<PackageVersionListItemFragment_version$key>(graphql`
        fragment PackageVersionListItemFragment_version on PackageVersion {
            metadata {
                createdAt
            }
            id
            version
            createdBy
            latest
            status
        }
    `, props.fragmentRef);

    return (
        // The row selects the version it describes, so the page shows that version's files and details.
        <ListItemButton
            component="div"
            onClick={() => onSelect?.(data.id)}
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
                primary={<Box display="flex" alignItems="center" gap={1}>
                    <Typography>{data.version}</Typography>
                    {data.latest && <Chip size="small" color="secondary" label="latest" />}
                    {data.status !== 'UPLOADED' && <Chip size="small" variant="outlined" color={data.status === 'ERRORED' ? 'error' : 'warning'} label={data.status} />}
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
        </ListItemButton>
    );
}

export default PackageVersionListItem;
