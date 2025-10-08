import { Avatar, Link, ListItemButton, ListItemText, Tooltip } from '@mui/material';
import teal from '@mui/material/colors/teal';
import graphql from 'babel-plugin-relay/macro';
import { useMemo } from 'react';
import { useFragment } from 'react-relay/hooks';
import { useNavigate } from 'react-router-dom';
import { HomeWorkspaceListItemFragment_workspace$key } from "./__generated__/HomeWorkspaceListItemFragment_workspace.graphql";

interface Props {
    fragmentRef: HomeWorkspaceListItemFragment_workspace$key
    last?: boolean
}

function HomeWorkspaceListItem({ fragmentRef, last }: Props) {
    const navigate = useNavigate();

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
            onClick={() => navigate(`/groups/${data.fullPath}`)}
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
                        <Link
                            underline="hover"
                            fontWeight={500}
                            variant="body2"
                            color="textPrimary"
                            sx={{
                                wordWrap: 'break-word'
                            }}
                        >
                            {formattedWorkspacePath}
                        </Link>
                    </Tooltip>
                }
            />
        </ListItemButton>
    );
}

export default HomeWorkspaceListItem;
