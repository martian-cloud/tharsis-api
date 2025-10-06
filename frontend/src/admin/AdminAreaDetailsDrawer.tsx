import { Box, List, ListItemButton, ListItemIcon, ListItemText, useTheme } from "@mui/material";
import { AnnouncementIcon, RunnerIcon, SettingsIcon, UserIcon } from "../common/Icons";
import Drawer from '../common/Drawer';
import { useLocation, useNavigate } from 'react-router-dom';

const LIST_ITEMS = [
    { route: 'users', label: 'Users', icon: <UserIcon /> },
    { route: 'runners', label: 'Runner Agents', icon: <RunnerIcon /> },
    { route: 'announcements', label: 'Announcements', icon: <AnnouncementIcon /> },
    { route: 'system_settings', label: 'System Settings', icon: <SettingsIcon /> },
]

const DRAWER_WIDTH = 240;

function AdminAreaDetailsDrawer() {
    const theme = useTheme();
    const navigate = useNavigate();
    const location = useLocation();
    const route = location.pathname as string;

    return (
        <Drawer
            width={DRAWER_WIDTH}
            mobileWidth={`calc(${theme.spacing(7)} + 1px)`}
            variant="permanent"
        >
            <Box>
                <List>
                    {LIST_ITEMS.map(item => (
                        <ListItemButton
                            key={item.route}
                            selected={route.includes(item.route)}
                            onClick={() => navigate(`/admin/${item.route}`)}
                        >
                            <ListItemIcon>{item.icon}</ListItemIcon>
                            <ListItemText>{item.label}</ListItemText>
                        </ListItemButton>)
                    )}
                </List>
            </Box>
        </Drawer>
    );
}

export default AdminAreaDetailsDrawer
