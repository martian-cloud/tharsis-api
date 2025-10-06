import { useState, useMemo } from 'react';
import NotificationsOnIcon from '@mui/icons-material/NotificationsNone';
import { Box, Button, Divider, Menu, MenuItem, Typography, Tooltip } from '@mui/material';
import { NotificationsOff as NotificationsOffIcon, Settings as SettingsIcon, ArrowDropDown as ArrowDropDownIcon } from '@mui/icons-material';
import { useFragment } from 'react-relay/hooks';
import Link from '../routes/Link';
import graphql from 'babel-plugin-relay/macro';
import NotificationPreferenceDialog from './NotificationPreferenceDialog';
import { NotificationButtonFragment_notificationPreference$key, UserNotificationPreferenceScope } from './__generated__/NotificationButtonFragment_notificationPreference.graphql';

export type Preference = {
    readonly customEvents: {
        readonly failedRun: boolean;
    } | null | undefined;
    readonly global: boolean;
    readonly inherited: boolean;
    readonly namespacePath: string | null | undefined;
    readonly scope: UserNotificationPreferenceScope;
}

interface NotificationOption {
    label: string;
    description: string;
}

export const notificationOptions: NotificationOption[] = [
    {
        label: 'ALL',
        description: 'Receive all notifications for all events in the system'
    },
    {
        label: 'PARTICIPATE',
        description: 'Receive notifications only for events you participate in'
    },
    {
        label: 'CUSTOM',
        description: 'Receive notifications for a custom list of events'
    },
    {
        label: 'NONE',
        description: 'Do not receive any notifications'
    }
];

interface InheritedMessageProps {
    preferenceData: Preference;
}

export const InheritedMessage = ({ preferenceData }: InheritedMessageProps) => (
    <Typography
        variant="caption"
        sx={{
            fontStyle: 'italic'
        }}
    >
        Inherited from {preferenceData.global ? 'global preference' : (
            <Box
                component="span"
                sx={{
                    display: 'inline-block',
                    maxWidth: '200px',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                    verticalAlign: 'bottom'
                }}
            >
                <Link
                    to={`/groups/${preferenceData.namespacePath}`}
                    children={preferenceData.namespacePath}
                />
            </Box>
        )}
    </Typography>
);

interface Props {
    path: string | null;
    isGlobalPreference?: boolean;
    fragmentRef: NotificationButtonFragment_notificationPreference$key;
}

function NotificationButton({ path, fragmentRef, isGlobalPreference = false }: Props) {
    const [menuAnchorEl, setMenuAnchorEl] = useState<Element | null>(null);
    const [showDialog, setShowDialog] = useState(false);
    const [localPreference, setLocalPreference] = useState<Preference| null>(null);

    const data = useFragment<NotificationButtonFragment_notificationPreference$key>(
        graphql`
            fragment NotificationButtonFragment_notificationPreference on UserNotificationPreference {
                scope
                inherited
                namespacePath
                global
                customEvents {
                    failedRun
                }
            }
        `,
        fragmentRef
    );

    // Merge local state with data from fragment
    const preferenceData = localPreference || data;

    const onDialogOpen = () => {
        setMenuAnchorEl(null);
        setShowDialog(true);
    };

    const onDialogClose = () => {
        setShowDialog(false);
        setMenuAnchorEl(null);
    };

    const notificationIcon = useMemo(() => {
        if (preferenceData.scope === 'NONE') {
            return <NotificationsOffIcon  />;
        }
        return <NotificationsOnIcon />;
    }, [preferenceData]);

    return (
        <Box>
            <Tooltip title={`Notifications: ${preferenceData.scope}`}>
                <Button
                    size="small"
                    aria-label="notification button"
                    aria-haspopup="menu"
                    variant="outlined"
                    color="info"
                    onClick={(event) => setMenuAnchorEl(event.currentTarget)}
                >
                    {notificationIcon}
                    {isGlobalPreference && (
                        <Typography variant="body2" sx={{ mx: 1 }}>
                            {preferenceData.scope}
                        </Typography>
                    )}
                    <ArrowDropDownIcon fontSize="small" />
                </Button>
            </Tooltip>
            <Menu
                id="notification-menu"
                anchorEl={menuAnchorEl}
                open={Boolean(menuAnchorEl)}
                onClose={() => setMenuAnchorEl(null)}
                anchorOrigin={{
                    vertical: 'bottom',
                    horizontal: 'right',
                }}
                transformOrigin={{
                    vertical: 'top',
                    horizontal: 'right',
                }}
            >
                <Box sx={{ pt: 2 }}>
                    <Box
                        sx={{
                            display: 'flex',
                            width: '100%',
                            flexDirection: 'column',
                            alignItems: 'flex-start',
                            px: 2,
                            pb: 2,
                        }}
                    >
                        <Typography
                            fontWeight="bold">
                            {preferenceData.scope}
                        </Typography>
                        <Typography
                            variant="caption"
                            color="textSecondary"
                        >
                            {notificationOptions.find(opt => opt.label === preferenceData.scope)?.description}
                        </Typography>
                        {preferenceData.inherited && (
                            <InheritedMessage preferenceData={preferenceData} />
                        )}
                    </Box>
                    <Divider />
                    <MenuItem
                        onClick={onDialogOpen}
                        sx={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: 1,
                            p: 2
                        }}
                    >
                        <SettingsIcon fontSize="small" />
                        <Typography>Change Preference</Typography>
                    </MenuItem>
                </Box>
            </Menu>
            {showDialog && <NotificationPreferenceDialog
                path={path}
                onClose={onDialogClose}
                isGlobalPreference={isGlobalPreference}
                preferenceData={preferenceData}
                onPreferenceUpdated={(pref: Preference) => setLocalPreference(pref)}
            />}
        </Box>
    );
}

export default NotificationButton;
