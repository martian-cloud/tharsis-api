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
            apply_queuing: string;
            applying: string;
            canceled: string;
            discarded: string;
            errored: string;
            pending: string;
            plan_queued: string;
            plan_queuing: string;
            planned: string;
            planned_and_finished: string;
            planning: string;
            pre_plan_queuing: string;
            pre_plan_running: string;
            pre_plan_awaiting_decision: string;
            pre_plan_completed: string;
            post_plan_running: string;
            post_plan_awaiting_decision: string;
            pre_apply_queuing: string;
            pre_apply_running: string;
            pre_apply_awaiting_decision: string;
            pre_apply_completed: string;
            post_apply_running: string;
            created: string;
            finished: string;
            running: string;
            queued: string;
            skipped: string;
            destroy: string;
            unknown: string;
            awaiting_decision: string;
        };
        jobStatus: {
            queued: string;
            pending: string;
            running: string;
            failed: string;
            canceled: string;
            canceling: string;
            finished: string;
        };
        planDiff: {
            create: string;
            delete: string;
            update: string;
            import: string;
            drift: string;
            read: string;
        };
        checkResult: {
            PASS: string;
            FAIL: string;
            ERROR: string;
            UNKNOWN: string;
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
        jobStatus?: Palette['jobStatus'];
        planDiff?: Palette['planDiff'];
        checkResult?: Palette['checkResult'];
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
            apply_queued: '#8ba3c7',
            apply_queuing: '#8ba3c7',
            applying: '#60a5fa',
            canceled: '#f87171',
            discarded: '#cbd5e1',
            errored: '#f87171',
            pending: '#8ba3c7',
            plan_queued: '#8ba3c7',
            plan_queuing: '#8ba3c7',
            planned: '#c084fc',
            planned_and_finished: '#34d399',
            planning: '#60a5fa',
            pre_plan_queuing: '#8ba3c7',
            pre_plan_running: '#60a5fa',
            pre_plan_awaiting_decision: '#fbbf24',
            pre_plan_completed: '#34d399',
            post_plan_running: '#60a5fa',
            post_plan_awaiting_decision: '#fbbf24',
            pre_apply_queuing: '#8ba3c7',
            pre_apply_running: '#60a5fa',
            pre_apply_awaiting_decision: '#fbbf24',
            pre_apply_completed: '#34d399',
            post_apply_running: '#60a5fa',
            created: '#8ba3c7',
            finished: '#34d399',
            running: '#60a5fa',
            queued: '#8ba3c7',
            skipped: '#cbd5e1',
            destroy: '#f87171',
            unknown: '#94a3b8',
            awaiting_decision: '#fbbf24',
        },
        jobStatus: {
            queued: '#8ba3c7',
            pending: '#8ba3c7',
            running: '#60a5fa',
            failed: '#f87171',
            canceled: '#f87171',
            canceling: '#fb923c',
            finished: '#34d399',
        },
        planDiff: {
            create: '#34d399',
            delete: '#f87171',
            update: '#c084fc',
            import: '#60a5fa',
            drift: '#fbbf24',
            read: '#5eead4',
        },
        checkResult: {
            PASS: '#34d399',
            FAIL: '#f87171',
            ERROR: '#fbbf24',
            UNKNOWN: '#9ca3af',
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
        background: {
            default: '#121212',
            paper: '#1e1e1e',
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
        MuiAppBar: {
            styleOverrides: {
                root: ({ theme }) => ({
                    backgroundColor: theme.palette.background.default,
                }),
            },
        },
        MuiDrawer: {
            styleOverrides: {
                paper: ({ theme }) => ({
                    backgroundColor: theme.palette.background.default,
                }),
            },
        },
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
