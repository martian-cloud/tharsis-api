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

    interface Palette {
        runStatus: {
            applied: string;
            apply_queued: string;
            applying: string;
            canceled: string;
            errored: string;
            pending: string;
            plan_queued: string;
            planned: string;
            planned_and_finished: string;
            planning: string;
            created: string;
            finished: string;
            running: string;
            queued: string;
            destroy: string;
            unknown: string;
        };
        planDiff: {
            create: string;
            delete: string;
            update: string;
            import: string;
            drift: string;
            read: string;
        };
        avatar: {
            default: string;
            serviceAccount: string;
        };
        announcement: {
            info: { main: string; dark: string; light: string };
            error: { main: string; dark: string; light: string };
            warning: { main: string; dark: string; light: string };
            success: { main: string; dark: string; light: string };
        };
    }

    interface PaletteOptions {
        runStatus?: Palette['runStatus'];
        planDiff?: Palette['planDiff'];
        avatar?: Palette['avatar'];
        announcement?: Palette['announcement'];
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
        success: {
            main: '#34d399',
        },
        error: {
            main: '#f87171',
        },
        warning: {
            main: '#fbbf24',
        },
        info: {
            main: 'rgba(255,255,255,0.7)'
        },
        runStatus: {
            applied: '#34d399',
            apply_queued: '#fbbf24',
            applying: '#60a5fa',
            canceled: '#f87171',
            errored: '#f87171',
            pending: '#fbbf24',
            plan_queued: '#fbbf24',
            planned: '#60a5fa',
            planned_and_finished: '#34d399',
            planning: '#60a5fa',
            created: '#94a3b8',
            finished: '#34d399',
            running: '#60a5fa',
            queued: '#fbbf24',
            destroy: '#f87171',
            unknown: '#94a3b8',
        },
        planDiff: {
            create: '#34d399',
            delete: '#f87171',
            update: '#c084fc',
            import: '#60a5fa',
            drift: '#fbbf24',
            read: '#5eead4',
        },
        avatar: {
            default: teal[200],
            serviceAccount: '#d8b4fe',
        },
        announcement: {
            info: { main: teal[300], dark: teal[500], light: teal[200] },
            error: { main: '#f44336', dark: '#d32f2f', light: '#e91e63' },
            warning: { main: '#ff6d00', dark: '#e65100', light: '#f9a825' },
            success: { main: '#0984e3', dark: '#1a73e8', light: '#6c5ce7' },
        },
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
