import { useState } from 'react';
import { Box, Button, Menu, SxProps, Typography, useTheme } from '@mui/material';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { atomDark as prismTheme } from 'react-syntax-highlighter/dist/esm/styles/prism';
import CopyButton from './CopyButton';
import ArrowDropDownIcon from '@mui/icons-material/ArrowDropDown';

interface Props {
    trn: string;
    sx?: SxProps;
    size?: 'small' | 'medium' | 'large';
}

function TRNButton({ sx, trn, size }: Props) {
    const [menuAnchorEl, setMenuAnchorEl] = useState<Element | null>(null);
    const theme = useTheme();

    return (
        <Box>
            <Button
                sx={{ ...sx }}
                size={size}
                aria-label='trn button'
                aria-haspopup="menu"
                variant='outlined'
                color="info"
                onClick={(event) => setMenuAnchorEl(event.currentTarget)}
            >
                TRN
                <ArrowDropDownIcon fontSize="small" />
            </Button>
            <Menu
                id="trn-menu"
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
                <Box sx={{ pl: 2, pr: 2, pt: 1, pb: 1 }}>
                    <Typography sx={{ mb: 1 }}>Tharsis Resource Name</Typography>
                    <Box sx={{
                        display: "flex",
                        border: 1,
                        borderRadius: 1,
                        borderColor: 'divider',
                        alignItems: 'center',
                    }}>
                        <SyntaxHighlighter
                            style={prismTheme}
                            customStyle={{
                                fontSize: 14,
                                margin: '0px',
                                width: '100%',
                                borderRight: `1px solid ${theme.palette.divider}`
                            }}
                        >
                            {trn}
                        </SyntaxHighlighter>
                        <CopyButton
                            data={trn}
                            toolTip="Copy TRN"
                        />
                    </Box>
                </Box>
            </Menu>
        </Box>
    );
}

export default TRNButton;
