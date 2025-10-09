import { Button } from '@mui/material';
import CssBaseline from '@mui/material/CssBaseline';
import { ThemeProvider } from "@mui/material/styles";
import { SnackbarProvider } from 'notistack';
import React from 'react';
import { CookiesProvider } from 'react-cookie';
import { RelayEnvironmentProvider } from 'react-relay';
import { BrowserRouter, useLocation } from 'react-router-dom';
import Root from './Root';
import AuthServiceContext from './auth/AuthServiceContext';
import AuthenticationService from './auth/AuthenticationService';
import LoginPage from './login/LoginPage';
import theme from './theme';

function MainContent() {
    const location = useLocation();
    return location.pathname === '/login' ? <LoginPage /> : <Root />;
}


interface Props {
    authService: AuthenticationService
    environment: any
}

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
                                <MainContent />
                            </SnackbarProvider>
                        </ThemeProvider>
                    </AuthServiceContext.Provider>
                </RelayEnvironmentProvider>
            </BrowserRouter>
        </CookiesProvider>
    );
}

export default App;
