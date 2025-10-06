import Tab, { TabProps } from '@mui/material/Tab';
import React from 'react';
import { Link as RouterLink, LinkProps as RouterLinkProps } from 'react-router-dom';

function Link(props: RouterLinkProps & TabProps<any, RouterLinkProps>) {
    const { replace, ...other } = props;
    return (
        <Tab
            {...other}
            replace={replace ?? true}
            component={RouterLink}
        />
    );
}

export default Link