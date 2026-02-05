import { createTheme } from "@mui/material";
import { teal } from '@mui/material/colors';

// TODO: In a future story this will be configurable via settings
const mode = 'dark' as any

declare module '@mui/material/Chip' {
    interface ChipPropsSizeOverrides {
        xs: true;
    }
}

declare module '@mui/material/styles' {
    interface TypographyVariants {
        code: React.CSSProperties;
    }

    interface TypographyVariantsOptions {
        code?: React.CSSProperties;
    }
}

declare module '@mui/material/Typography' {
    interface TypographyPropsVariantOverrides {
        code: true;
    }
}

export default createTheme({
    palette: {
        mode,
        primary: {
            main: mode === 'dark' ? teal[300] : teal[500]
        },
        secondary: {
            main: '#29b6f6'
        },
        info: {
            main: 'rgba(255,255,255,0.7)'
        }
    },
    typography: {
        fontFamily: [
            '-apple-system',
            'BlinkMacSystemFont',
            '"Segoe UI"',
            'Roboto',
            '"Helvetica Neue"',
            'Arial',
            'sans-serif',
            '"Apple Color Emoji"',
            '"Segoe UI Emoji"',
            '"Segoe UI Symbol"',
        ].join(','),
        code: {
            fontFamily: 'ui-monospace,SFMono-Regular,SF Mono,Menlo,Consolas,Liberation Mono,monospace',
            fontSize: `0.85rem`
        }
    },
    components: {
        MuiChip: {
            variants: [
                {
                    props: { size: 'xs' },
                    style: {
                        fontSize: '0.75rem',
                        lineHeight: '1rem',
                        height: '20px',
                        borderRadius: '0.25rem'
                    }
                }
            ]
        }
    }
});
