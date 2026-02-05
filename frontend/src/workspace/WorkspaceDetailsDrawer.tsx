import {
    Avatar,
    Box,
    List,
    ListItem,
    ListItemAvatar,
    ListItemIcon,
    ListItemText,
    ListItemButton,
    Typography,
    useMediaQuery,
    useTheme
} from '@mui/material';
import StateIcon from '@mui/icons-material/InsertDriveFileOutlined';
import MembersIcon from '@mui/icons-material/PeopleOutline';
import SettingsIcon from '@mui/icons-material/SettingsOutlined';
import ActivityIcon from '@mui/icons-material/TimelineOutlined';
import VariablesIcon from '@mui/icons-material/WindowOutlined';
import { teal } from '@mui/material/colors';
import { Link } from 'react-router-dom';
import Drawer from '../common/Drawer';
import { AccountLockOutline as ManagedIdentityIcon, ContentDuplicate as ProviderMirrorIcon, RocketLaunchOutline as RunIcon } from 'mdi-material-ui';

interface Props {
    workspacePath: string
    workspaceName: string
    route: string
}

const DRAWER_WIDTH = 240;

const LIST_ITEMS = [
    { route: 'activity', label: 'Activity', icon: <ActivityIcon /> },
    { route: 'runs', label: 'Runs', icon: <RunIcon /> },
    { route: 'variables', label: 'Variables', icon: <VariablesIcon /> },
    { route: 'state_versions', label: 'State Versions', icon: <StateIcon /> },
    { route: 'managed_identities', label: 'Assigned Managed Identities', icon: <ManagedIdentityIcon /> },
    { route: 'provider_mirror', label: 'Provider Mirror', icon: <ProviderMirrorIcon /> },
    { route: 'members', label: 'Members', icon: <MembersIcon /> },
    { route: 'settings', label: 'Settings', icon: <SettingsIcon /> }
];

function WorkspaceDetailsDrawer(props: Props) {
    const { route, workspaceName, workspacePath } = props;
    const theme = useTheme();
    const fullSize = useMediaQuery(theme.breakpoints.up('md'));

    return (
        <Drawer
            width={DRAWER_WIDTH}
            mobileWidth={`calc(${theme.spacing(7)} + 1px)`}
            variant="permanent"
        >
            <Box>
                <List>
                    {fullSize && <ListItem dense>
                        <Typography variant="subtitle2" color="textSecondary">Workspace</Typography>
                    </ListItem>}
                    <ListItemButton
                        component={Link}
                        to={`/groups/${workspacePath}`}
                    >
                        <ListItemAvatar>
                            <Avatar sx={{ width: 24, height: 24, bgcolor: teal[200] }} variant="rounded">{workspaceName[0].toUpperCase()}</Avatar>
                        </ListItemAvatar>
                        {fullSize && <ListItemText sx={{ wordWrap: 'break-word' }} primary={workspaceName} />}
                    </ListItemButton>
                    {LIST_ITEMS.map(item => (
                        <ListItemButton
                            key={item.route}
                            selected={route === item.route}
                            component={Link}
                            to={`/groups/${workspacePath}/-/${item.route}`}>
                            <ListItemIcon sx={{ mt: 0.5, mb: 0.5 }}>
                                {item.icon}
                            </ListItemIcon>
                            {fullSize && <ListItemText primary={item.label} />}
                        </ListItemButton>
                    ))}
                </List>
            </Box>
        </Drawer>
    );
}

export default WorkspaceDetailsDrawer;
