import { Button } from '@mui/material';
import CssBaseline from '@mui/material/CssBaseline';
import teal from '@mui/material/colors/teal';
import { ThemeProvider, createTheme } from "@mui/material/styles";
import { SnackbarProvider } from 'notistack';
import React from 'react';
import { CookiesProvider } from 'react-cookie';
import { RelayEnvironmentProvider } from 'react-relay';
import { BrowserRouter } from 'react-router-dom';
import Root from './Root';
import AuthServiceContext from './auth/AuthServiceContext';
import AuthenticationService from './auth/AuthenticationService';


interface Props {
    authService: AuthenticationService
    environment: any
}

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

const theme = createTheme({
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

function App(props: Props) {
    // Add dismiss action to all snackbars
    const notistackRef = React.createRef<SnackbarProvider>();
    const onClickDismiss = (key: any) => () => {
        notistackRef.current?.closeSnackbar(key);
    }

    return (
        <CookiesProvider>
            <BrowserRouter>
                <RelayEnvironmentProvider environment={props.environment}>
                    <AuthServiceContext.Provider value={props.authService}>
                        <ThemeProvider theme={theme}>
                            <SnackbarProvider
                                ref={notistackRef}
                                action={(key) => (
                                    <Button onClick={onClickDismiss(key)} color="inherit">
                                        Dismiss
                                    </Button>
                                )}
                            >
                                <CssBaseline />
                                <Root />
                            </SnackbarProvider>
                        </ThemeProvider>
                    </AuthServiceContext.Provider>
                </RelayEnvironmentProvider>
            </BrowserRouter>
        </CookiesProvider>
    );
}

export default App;
