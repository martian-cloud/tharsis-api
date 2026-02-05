import { Avatar, ListItemButton, ListItemText, Tooltip, Typography } from '@mui/material';
import { teal } from '@mui/material/colors';
import graphql from 'babel-plugin-relay/macro';
import { useMemo } from 'react';
import { useFragment } from 'react-relay/hooks';
import { Link } from 'react-router-dom';
import { HomeWorkspaceListItemFragment_workspace$key } from "./__generated__/HomeWorkspaceListItemFragment_workspace.graphql";

interface Props {
    fragmentRef: HomeWorkspaceListItemFragment_workspace$key
    last?: boolean
}

function HomeWorkspaceListItem({ fragmentRef, last }: Props) {

    const data = useFragment(graphql`
        fragment HomeWorkspaceListItemFragment_workspace on Workspace
        {
            name
            fullPath
        }
    `, fragmentRef);

    const formattedWorkspacePath = useMemo(() => {
        const path = data.fullPath;
        const pathParts = path.split('/');

        return pathParts.length > 3 ? `${pathParts[0]} / ... / ${pathParts[pathParts.length - 1]}` : path;

    }, [data.fullPath]);

    return (
        <ListItemButton
            dense
            component={Link}
            to={`/groups/${data.fullPath}`}
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
                sx={{ overflow: "hidden" }}
                primary={
                    <Tooltip title={data.fullPath}>
                        <Typography
                            fontWeight={500}
                            variant="body2"
                            color="textPrimary"
                            sx={{
                                wordWrap: 'break-word',
                                textDecoration: 'underline',
                                textDecorationColor: 'transparent',
                                '&:hover': {
                                    textDecorationColor: 'currentColor'
                                }
                            }}
                        >
                            {formattedWorkspacePath}
                        </Typography>
                    </Tooltip>
                }
            />
        </ListItemButton>
    );
}

export default HomeWorkspaceListItem;
