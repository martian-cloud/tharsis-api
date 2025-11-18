
import DropdownIcon from '@mui/icons-material/ArrowDropDown';
import { Button, Menu, MenuItem } from '@mui/material';
import Box from '@mui/material/Box';
import { Link } from 'react-router-dom';
import React, { useState } from 'react';

function RegistryMenu() {
    const [menuAnchorEl, setMenuAnchorEl] = useState<null | HTMLElement>(null);

    function onMenuOpen(event: React.MouseEvent<HTMLButtonElement>) {
        setMenuAnchorEl(event.currentTarget);
    }

    function onMenuClose() {
        setMenuAnchorEl(null);
    }

    return (
        <Box>
            <Button color="inherit" sx={{ textTransform: "none", fontWeight: 600 }} onClick={onMenuOpen}>Registry <DropdownIcon /></Button>
            <Menu
                id="registry-menu"
                anchorEl={menuAnchorEl}
                keepMounted
                open={Boolean(menuAnchorEl)}
                onClose={onMenuClose}
            >
                <MenuItem component={Link} to="/module-registry" onClick={onMenuClose}>Modules</MenuItem>
                <MenuItem component={Link} to="/provider-registry" onClick={onMenuClose}>Providers</MenuItem>
            </Menu>
        </Box>
    );
}

export default RegistryMenu;
