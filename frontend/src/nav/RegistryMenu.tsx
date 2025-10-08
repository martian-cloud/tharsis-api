
import DropdownIcon from '@mui/icons-material/ArrowDropDown';
import { Button, Menu, MenuItem } from '@mui/material';
import Box from '@mui/material/Box';
import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';

function RegistryMenu() {
    const navigate = useNavigate();
    const [menuAnchorEl, setMenuAnchorEl] = useState<null | HTMLElement>(null);

    function onMenuOpen(event: React.MouseEvent<HTMLButtonElement>) {
        setMenuAnchorEl(event.currentTarget);
    }

    function onMenuClose() {
        setMenuAnchorEl(null);
    }

    function onNavigate(path: string) {
        onMenuClose();
        navigate(path);
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
                <MenuItem onClick={() => onNavigate('/module-registry')}>Modules</MenuItem>
                <MenuItem onClick={() => onNavigate('/provider-registry')}>Providers</MenuItem>
            </Menu>
        </Box>
    );
}

export default RegistryMenu;
