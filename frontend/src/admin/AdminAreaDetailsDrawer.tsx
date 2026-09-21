import BlockOutlinedIcon from '@mui/icons-material/BlockOutlined';
import EmailOutlinedIcon from '@mui/icons-material/EmailOutlined';
import SpeedOutlinedIcon from '@mui/icons-material/SpeedOutlined';
import TerminalOutlinedIcon from '@mui/icons-material/TerminalOutlined';
import TuneOutlinedIcon from '@mui/icons-material/TuneOutlined';
import { Avatar, Box, List, ListItemButton, ListItemIcon, ListItemText, ListSubheader, Typography, useMediaQuery, useTheme } from '@mui/material';
import { alpha } from '@mui/material/styles';
import { Link, useLocation } from 'react-router-dom';
import { AnnouncementIcon, RunnerIcon, SettingsIcon, UserIcon } from '../common/Icons';
import Drawer from '../common/Drawer';

const DRAWER_WIDTH = 224;

const NAV_SECTIONS = [
    {
        items: [
            { route: 'users', label: 'Users', icon: <UserIcon /> },
            { route: 'runners', label: 'Runner Agents', icon: <RunnerIcon /> },
        ],
    },
    {
        label: 'Communication',
        items: [
            { route: 'announcements', label: 'Announcements', icon: <AnnouncementIcon /> },
            { route: 'email_outbox', label: 'Email Outbox', icon: <EmailOutlinedIcon /> },
            { route: 'email_suppressions', label: 'Email Suppressions', icon: <BlockOutlinedIcon /> },
        ],
    },
    {
        label: 'System',
        items: [
            { route: 'system_settings', label: 'System Settings', icon: <SettingsIcon /> },
            { route: 'configuration', label: 'API Configuration', icon: <TuneOutlinedIcon /> },
            { route: 'resource_limits', label: 'Resource Limits', icon: <SpeedOutlinedIcon /> },
            { route: 'logs', label: 'API Logs', icon: <TerminalOutlinedIcon /> },
        ],
    },
];

function AdminAreaDetailsDrawer() {
    const theme = useTheme();
    const location = useLocation();
    const fullSize = useMediaQuery(theme.breakpoints.up('md'));

    // The active section is the first path segment after /admin/.
    const segment = location.pathname.replace(/^\/admin\/?/, '').split('/')[0];

    const navItemSx = {
        borderRadius: '7px',
        mx: theme.spacing(1),
        py: theme.spacing(0.375),
        px: theme.spacing(1.25),
        minHeight: 0,
        my: fullSize ? theme.spacing(0.125) : theme.spacing(2),
        position: 'relative',
        justifyContent: fullSize ? 'flex-start' : 'center',
        '& .MuiListItemIcon-root': {
            minWidth: 0,
            mr: fullSize ? theme.spacing(1.25) : 0,
            color: fullSize ? theme.palette.text.secondary : 'inherit',
            '& svg': { width: fullSize ? 16 : undefined, height: fullSize ? 16 : undefined },
        },
        '& .MuiListItemText-primary': {
            fontSize: theme.typography.body2.fontSize,
            fontWeight: 500,
            lineHeight: 1.5,
        },
        '&:hover': { background: theme.palette.action.hover },
        '&.Mui-selected': {
            background: alpha(theme.palette.primary.main, 0.08),
            '&::before': {
                content: '""',
                position: 'absolute',
                left: 0,
                top: theme.spacing(0.625),
                bottom: theme.spacing(0.625),
                width: '3px',
                borderRadius: '0 3px 3px 0',
                background: theme.palette.primary.main,
            },
            '& .MuiListItemIcon-root': { color: theme.palette.primary.main },
            '& .MuiListItemText-primary': { color: theme.palette.text.primary, fontWeight: 500 },
            '&:hover': { background: alpha(theme.palette.primary.main, 0.12) },
        },
    } as const;

    return (
        <Drawer
            width={DRAWER_WIDTH}
            mobileWidth={`calc(${theme.spacing(7)} + 1px)`}
            variant="permanent"
        >
            {/* Admin header */}
            <Box sx={{ px: theme.spacing(1), pt: theme.spacing(1), pb: theme.spacing(0.75), borderBottom: `1px solid ${theme.palette.divider}`, mb: theme.spacing(0.25) }}>
                <ListItemButton
                    component={Link}
                    to="/admin"
                    sx={{
                        py: theme.spacing(0.75),
                        px: theme.spacing(1),
                        borderRadius: '8px',
                        minHeight: 0,
                        gap: theme.spacing(1.25),
                        justifyContent: fullSize ? 'flex-start' : 'center',
                        '&:hover': { background: theme.palette.action.hover },
                    }}
                >
                    <Avatar
                        sx={{
                            width: 28,
                            height: 28,
                            borderRadius: '6px',
                            background: `linear-gradient(135deg, ${theme.palette.primary.dark}, ${theme.palette.primary.main})`,
                            fontSize: theme.typography.caption.fontSize,
                            fontWeight: 600,
                            color: theme.palette.primary.contrastText,
                            flexShrink: 0,
                        }}
                    >
                        A
                    </Avatar>
                    {fullSize && (
                        <Box sx={{ flex: 1, minWidth: 0 }}>
                            <Typography variant="subtitle2" component="div" sx={{ color: theme.palette.text.primary, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', lineHeight: 1.3 }}>
                                Admin
                            </Typography>
                            <Typography variant="caption" component="div" sx={{ color: theme.palette.text.disabled, mt: theme.spacing(0.125), lineHeight: 1 }}>
                                Administration
                            </Typography>
                        </Box>
                    )}
                </ListItemButton>
            </Box>

            {/* Nav sections */}
            <List disablePadding sx={{ pt: theme.spacing(0.25), pb: theme.spacing(1) }}>
                {NAV_SECTIONS.map((section, idx) => (
                    <Box key={section.label ?? idx}>
                        {fullSize && section.label && (
                            <ListSubheader
                                disableSticky
                                sx={{
                                    mt: 1,
                                    px: theme.spacing(2.25),
                                    py: 0,
                                    lineHeight: '28px',
                                    fontSize: theme.typography.caption.fontSize,
                                    fontWeight: 600,
                                    letterSpacing: '0.08em',
                                    color: theme.palette.text.secondary,
                                    bgcolor: 'transparent',
                                    textTransform: 'uppercase',
                                }}
                            >
                                {section.label}
                            </ListSubheader>
                        )}
                        {section.items.map(item => (
                            <ListItemButton
                                key={item.route}
                                selected={segment === item.route}
                                component={Link}
                                to={`/admin/${item.route}`}
                                sx={navItemSx}
                            >
                                <ListItemIcon>{item.icon}</ListItemIcon>
                                {fullSize && <ListItemText primary={item.label} />}
                            </ListItemButton>
                        ))}
                    </Box>
                ))}
            </List>
        </Drawer>
    );
}

export default AdminAreaDetailsDrawer;
