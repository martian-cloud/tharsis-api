import { Avatar, ListItemButton, ListItemText, Typography } from '@mui/material';
import teal from '@mui/material/colors/teal';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { Link } from 'react-router-dom';
import { HomeTeamListItemFragment_team$key } from "./__generated__/HomeTeamListItemFragment_team.graphql";

interface Props {
    fragmentRef: HomeTeamListItemFragment_team$key
    last?: boolean
}

function HomeTeamListItem({ fragmentRef, last }: Props) {

    const data = useFragment(graphql`
        fragment HomeTeamListItemFragment_team on Team
        {
            name
        }
    `, fragmentRef);

    return (
        <ListItemButton
            dense
            component={Link}
            to={`/teams/${data.name}`}
            divider={!last}
        >
            <Avatar
                sx={{
                    width: 24,
                    height: 24,
                    mr: 2,
                    bgcolor: teal[200]
                }}
                variant="rounded">{data.name[0].toUpperCase()}
            </Avatar>
            <ListItemText
                primary={<Typography
                    fontWeight={500}
                    variant="body2"
                    color="textPrimary"
                    sx={{
                        textDecoration: 'underline',
                        textDecorationColor: 'transparent',
                        '&:hover': {
                            textDecorationColor: 'currentColor'
                        }
                    }}
                >
                    {data.name}
                </Typography>} />
        </ListItemButton>
    );
}

export default HomeTeamListItem;
