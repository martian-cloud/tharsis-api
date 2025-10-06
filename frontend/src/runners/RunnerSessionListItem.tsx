import { Box, Chip, ListItem, Typography, useTheme } from '@mui/material';
import { green, grey } from '@mui/material/colors';
import graphql from 'babel-plugin-relay/macro';
import moment from 'moment';
import { useFragment } from 'react-relay/hooks';
import { RunnerSessionListItemFragment$key } from './__generated__/RunnerSessionListItemFragment.graphql';

interface Props {
    fragmentRef: RunnerSessionListItemFragment$key
    onClick: () => void
}

function RunnerSessionListItem({ fragmentRef, onClick }: Props) {
    const theme = useTheme();

    const data = useFragment(graphql`
        fragment RunnerSessionListItemFragment on RunnerSession {
            id
            lastContacted
            active
            internal
            errorCount
            metadata {
                updatedAt
            }
        }
    `, fragmentRef);

    return (
        <ListItem
            dense
            sx={{
                borderBottom: `1px solid ${theme.palette.divider}`,
                borderLeft: `1px solid ${theme.palette.divider}`,
                borderRight: `1px solid ${theme.palette.divider}`,
                '&:last-child': {
                    borderBottomLeftRadius: 4,
                    borderBottomRightRadius: 4
                }
            }}>
            <Box display="flex" alignItems="center" justifyContent="space-between" flex={1} padding={1}>
                <Box display="flex" alignItems="center">
                    <Box sx={{ width: 16, height: 16, borderRadius: '50%', backgroundColor: data.active ? green[400] : grey[400], mr: 2 }} />
                    <Box>
                        <Box display="flex" alignItems="center">
                            <Box sx={{ textDecoration: 'none', minWidth: 100 }}>
                                {`${data.id.substring(0, 8)}...`}
                            </Box>
                            {data.internal && <Chip sx={{ fontSize: 12, ml: 1 }} variant="outlined" label="internal" size="small" />}
                        </Box>
                        <Typography variant="body2" color="textSecondary">
                            {`last seen ${moment(data.lastContacted as moment.MomentInput).fromNow()}`}
                        </Typography>
                    </Box>
                </Box>
                {data.errorCount > 0 && (<Chip onClick={onClick} label={`${data.errorCount} error${data.errorCount === 1 ? '' : 's'}`} color="error" size="small" sx={{ marginRight: 1 }} />)}
            </Box>
        </ListItem>
    );
}

export default RunnerSessionListItem
