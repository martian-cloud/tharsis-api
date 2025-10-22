import { Launch } from '@mui/icons-material';
import { Link, List, ListItem, ListItemButton, ListItemText } from '@mui/material';
import Box from '@mui/material/Box';
import Divider from '@mui/material/Divider';
import IconButton from '@mui/material/IconButton';
import Popover from '@mui/material/Popover';
import Typography from '@mui/material/Typography';
import graphql from 'babel-plugin-relay/macro';
import React, { useContext, useState } from 'react';
import { useFragment } from 'react-relay/hooks';
import { useNavigate } from 'react-router-dom';
import AuthServiceContext from '../auth/AuthServiceContext';
import AuthenticationService from '../auth/AuthenticationService';
import Gravatar from '../common/Gravatar';
import config from '../common/config';
import { AccountMenuFragment$key } from './__generated__/AccountMenuFragment.graphql';
import AboutDialog from './AboutDialog';
import { ApiConfigContext } from '../ApiConfigContext';
import { UserContext } from '../UserContext';

interface Props {
    fragmentRef: AccountMenuFragment$key
}

function AccountMenu({ fragmentRef }: Props) {
    const navigate = useNavigate();
    const authService = useContext<AuthenticationService>(AuthServiceContext);
    const [showAboutDialog, setShowAboutDialog] = useState(false);
    const [menuAnchorEl, setMenuAnchorEl] = useState<null | HTMLElement>(null);
    const apiConfig = useContext(ApiConfigContext);
    const user = useContext(UserContext);

    const data = useFragment<AccountMenuFragment$key>(
        graphql`
        fragment AccountMenuFragment on Query
        {
            version {
                version
                dbMigrationVersion
                dbMigrationDirty
                buildTimestamp
            }
        }
        `, fragmentRef);

    function onMenuOpen(event: React.MouseEvent<HTMLButtonElement>) {
        setMenuAnchorEl(event.currentTarget);
    }

    function onMenuClose() {
        setMenuAnchorEl(null);
    }

    function onShowGraphiql() {
        onMenuClose();
        navigate('graphiql');
    }

    function onShowAdminArea() {
        onMenuClose();
        navigate('admin');
    }

    function onShowAboutDialog() {
        onMenuClose();
        setShowAboutDialog(true);
    }

    function onShowPreferences() {
        onMenuClose();
        navigate('preferences');
    }

    return (
        <div>
            <IconButton onClick={onMenuOpen}><Gravatar width={32} height={32} email={user.email} /></IconButton>
            <Popover
                id="account-menu"
                open={Boolean(menuAnchorEl)}
                anchorEl={menuAnchorEl}
                onClose={onMenuClose}
                anchorOrigin={{
                    vertical: 'bottom',
                    horizontal: 'center',
                }}
                transformOrigin={{
                    vertical: 'top',
                    horizontal: 'center',
                }}
            >
                <div>
                    <Box padding={2}>
                        <Typography>{user.username}</Typography>
                    </Box>
                    <Divider />
                    <List dense>
                        <ListItemButton onClick={onShowPreferences}>
                            <ListItemText primary="Preferences" />
                        </ListItemButton>
                        {user.admin && <ListItemButton>
                            <ListItemText onClick={onShowAdminArea}>
                                Admin Area
                            </ListItemText>
                        </ListItemButton>}
                        <ListItemButton onClick={onShowGraphiql}>
                            <ListItemText primary="GraphQL Editor" />
                        </ListItemButton>
                        <ListItem secondaryAction={
                            <IconButton LinkComponent={Link}
                                edge='end'
                                href={config.docsUrl}
                                target='_blank'
                                rel='noopener noreferrer'
                                disableRipple
                            >
                                <Launch fontSize='small' />
                            </IconButton>
                        }
                            disablePadding
                        >
                            <ListItemButton LinkComponent={Link} href={config.docsUrl} target='_blank' rel='noopener noreferrer' dense>
                                <ListItemText primary="Documentation" />
                            </ListItemButton>
                        </ListItem>
                        {apiConfig.tharsisSupportUrl !== '' && <ListItem
                            secondaryAction={
                                <IconButton LinkComponent={Link}
                                    edge='end'
                                    href={apiConfig.tharsisSupportUrl}
                                    target='_blank'
                                    rel='noopener noreferrer'
                                    disableRipple
                                >
                                    <Launch fontSize='small' />
                                </IconButton>
                            }
                            disablePadding
                        >
                            <ListItemButton LinkComponent={Link} href={apiConfig.tharsisSupportUrl} target='_blank' rel='noopener noreferrer' dense>
                                <ListItemText primary="Support" />
                            </ListItemButton>
                        </ListItem>}
                        <ListItemButton>
                            <ListItemText onClick={onShowAboutDialog}>About Tharsis</ListItemText>
                        </ListItemButton>
                        <ListItemButton onClick={() => (authService.logout())}>
                            <ListItemText primary="Sign Out" />
                        </ListItemButton>
                    </List>
                </div>
            </Popover>
            {showAboutDialog && <AboutDialog
                version={data.version.version}
                buildTimestamp={data.version.buildTimestamp}
                dbMigrationVersion={data.version.dbMigrationVersion}
                dbMigrationDirty={data.version.dbMigrationDirty}
                onClose={() => setShowAboutDialog(false)}
            />}
        </div>
    );
}

export default AccountMenu;
