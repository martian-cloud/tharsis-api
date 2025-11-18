import { Launch } from '@mui/icons-material';
import { List, ListItem, ListItemButton, ListItemText } from '@mui/material';
import Box from '@mui/material/Box';
import Divider from '@mui/material/Divider';
import IconButton from '@mui/material/IconButton';
import Popover from '@mui/material/Popover';
import Typography from '@mui/material/Typography';
import graphql from 'babel-plugin-relay/macro';
import { Link } from 'react-router-dom';
import React, { useContext, useState } from 'react';
import { useFragment } from 'react-relay/hooks';
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

    function onShowAboutDialog() {
        onMenuClose();
        setShowAboutDialog(true);
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
                        <ListItemButton component={Link} to="/preferences" onClick={onMenuClose}>
                            <ListItemText primary="Preferences" />
                        </ListItemButton>
                        {user.admin && <ListItemButton component={Link} to="/admin" onClick={onMenuClose}>
                            <ListItemText primary="Admin Area" />
                        </ListItemButton>}
                        <ListItemButton component={Link} to="/graphiql" onClick={onMenuClose}>
                            <ListItemText primary="GraphQL Editor" />
                        </ListItemButton>
                        <ListItem secondaryAction={
                            <IconButton component={Link}
                                edge='end'
                                to={config.docsUrl}
                                target='_blank'
                                rel='noopener noreferrer'
                                disableRipple
                            >
                                <Launch fontSize='small' />
                            </IconButton>
                        }
                            disablePadding
                        >
                            <ListItemButton component={Link} to={config.docsUrl} target='_blank' rel='noopener noreferrer' dense>
                                <ListItemText primary="Documentation" />
                            </ListItemButton>
                        </ListItem>
                        {apiConfig.tharsisSupportUrl !== '' && <ListItem
                            secondaryAction={
                                <IconButton component={Link}
                                    edge='end'
                                    to={apiConfig.tharsisSupportUrl}
                                    target='_blank'
                                    rel='noopener noreferrer'
                                    disableRipple
                                >
                                    <Launch fontSize='small' />
                                </IconButton>
                            }
                            disablePadding
                        >
                            <ListItemButton component={Link} to={apiConfig.tharsisSupportUrl} target='_blank' rel='noopener noreferrer' dense>
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
