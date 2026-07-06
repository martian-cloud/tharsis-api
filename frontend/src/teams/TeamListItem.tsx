import { Box, ListItemButton, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { Link as RouterLink } from 'react-router-dom';
import { TeamListItemFragment_team$key } from './__generated__/TeamListItemFragment_team.graphql';

interface Props {
    fragmentRef: TeamListItemFragment_team$key
}

function TeamListItem({ fragmentRef }: Props) {
    const theme = useTheme();

    const data = useFragment<TeamListItemFragment_team$key>(graphql`
        fragment TeamListItemFragment_team on Team {
            name
            description
        }
    `, fragmentRef);

    return (
        <ListItemButton
            component={RouterLink}
            to={`/teams/${encodeURIComponent(data.name)}`}
            sx={{
                borderBottom: `1px solid ${theme.palette.divider}`,
                borderLeft: `1px solid ${theme.palette.divider}`,
                borderRight: `1px solid ${theme.palette.divider}`,
                '&:last-child': { borderBottomLeftRadius: 4, borderBottomRightRadius: 4 }
            }}
        >
            <Box sx={{ minWidth: 0 }}>
                <Typography sx={{ fontWeight: 500, wordBreak: 'break-word' }}>{data.name}</Typography>
                {data.description && <Typography variant="body2" color="textSecondary" sx={{ wordBreak: 'break-word' }}>
                    {data.description}
                </Typography>}
            </Box>
        </ListItemButton>
    );
}

export default TeamListItem;
