import MembersIcon from '@mui/icons-material/PeopleOutline';
import SettingsIcon from '@mui/icons-material/SettingsOutlined';
import ActivityIcon from '@mui/icons-material/TimelineOutlined';
import VariablesIcon from '@mui/icons-material/WindowOutlined';
import { Avatar, Box, List, ListItemButton, ListItemIcon, ListItemText, ListSubheader, Typography, useMediaQuery, useTheme } from '@mui/material';
import { alpha } from '@mui/material/styles';
import {
    ServerNetwork as FederatedRegistryIcon,
    KeyVariant as KeyIcon,
    AccountLockOutline as ManagedIdentityIcon,
    PackageVariantClosed as PackageIcon,
    ShieldLockOutline as PolicyIcon,
    ContentDuplicate as ProviderMirrorIcon,
    RocketLaunchOutline as RunIcon,
    RobotOutline as RunnersIcon,
    LanConnect as ServiceAccountIcon,
    CubeOutline as TerraformModuleIcon,
    SourceMerge as VCSProviderIcon,
    DeleteClockOutline as CleanupPolicyIcon
} from 'mdi-material-ui';
import { Link } from 'react-router-dom';
import Drawer from '../common/Drawer';

interface Props {
    groupPath: string
    groupName: string
    route: string
}

const DRAWER_WIDTH = 224;

const NAV_SECTIONS = [
    {
        items: [
            { route: 'activity', label: 'Activity', icon: <ActivityIcon /> },
            { route: 'runs', label: 'Runs', icon: <RunIcon /> },
            { route: 'variables', label: 'Variables', icon: <VariablesIcon /> },
        ],
    },
    {
        label: 'Security',
        items: [
            { route: 'managed_identities', label: 'Managed Identities', icon: <ManagedIdentityIcon /> },
            { route: 'service_accounts', label: 'Service Accounts', icon: <ServiceAccountIcon /> },
            { route: 'policies', label: 'Policies', icon: <PolicyIcon /> },
            { route: 'members', label: 'Members', icon: <MembersIcon /> },
        ],
    },
    {
        label: 'Registry',
        items: [
            { route: 'terraform_modules', label: 'Terraform Modules', icon: <TerraformModuleIcon /> },
            { route: 'packages', label: 'Packages', icon: <PackageIcon /> },
            { route: 'federated_registries', label: 'Federated Registries', icon: <FederatedRegistryIcon /> },
            { route: 'keys', label: 'GPG Keys', icon: <KeyIcon /> },
        ],
    },
    {
        label: 'Administration',
        items: [
            { route: 'runners', label: 'Runner Agents', icon: <RunnersIcon /> },
            { route: 'provider_mirror', label: 'Provider Mirror', icon: <ProviderMirrorIcon /> },
            { route: 'vcs_providers', label: 'VCS Providers', icon: <VCSProviderIcon /> },
            { route: 'cleanup_policies', label: 'Cleanup', icon: <CleanupPolicyIcon /> },
            { route: 'settings', label: 'Settings', icon: <SettingsIcon /> },
        ],
    },
];

function GroupDetailsDrawer(props: Props) {
    const { route, groupName, groupPath } = props;
    const theme = useTheme();
    const fullSize = useMediaQuery(theme.breakpoints.up('md'));

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
            {/* Group header */}
            <Box sx={{ px: theme.spacing(1), pt: theme.spacing(1), pb: theme.spacing(0.75), borderBottom: `1px solid ${theme.palette.divider}`, mb: theme.spacing(0.25) }}>
                <ListItemButton
                    component={Link}
                    to={`/groups/${groupPath}`}
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
                        {groupName[0].toUpperCase()}
                    </Avatar>
                    {fullSize && (
                        <Box sx={{ flex: 1, minWidth: 0 }}>
                            <Typography variant="subtitle2" component="div" sx={{ color: theme.palette.text.primary, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', lineHeight: 1.3 }}>
                                {groupName}
                            </Typography>
                            <Typography variant="caption" component="div" sx={{ color: theme.palette.text.disabled, mt: theme.spacing(0.125), lineHeight: 1 }}>
                                Group
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
                                selected={route === item.route}
                                component={Link}
                                to={`/groups/${groupPath}/-/${item.route}`}
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

export default GroupDetailsDrawer;
