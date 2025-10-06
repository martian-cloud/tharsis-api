import { Box } from '@mui/material';
import ListItem from '@mui/material/ListItem';
import { useTheme } from '@mui/material/styles';
import React from 'react';

interface Props {
    children: React.ReactNode
    nested?: boolean
    last?: boolean
}

function NestableTreeItem(props: Props) {
    const { children, nested, last } = props;
    const theme = useTheme();

    const containerStyle = {
        paddingLeft: 4,
        '&:before': {
            content: '""',
            background: theme.palette.divider,
            position: 'absolute',
            top: 0,
            left: 16,
            height: !last ? '100%' : 34,
            width: "1px"
        }
    } as any;

    const innerContainerStyle = { paddingTop: 1 } as any;
    if (nested) {
        innerContainerStyle['&:before'] = {
            content: '""',
            background: theme.palette.divider,
            position: 'absolute',
            top: '34px',
            left: 16,
            height: '1px',
            width: "16px"
        }
    }

    return (
        <ListItem sx={nested ? containerStyle : {}} disablePadding>
            <Box flex={1} sx={innerContainerStyle}>
                {children}
            </Box>
        </ListItem>
    )
}

export default NestableTreeItem
