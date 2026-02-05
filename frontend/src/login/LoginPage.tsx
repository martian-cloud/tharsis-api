import { Alert, Box, Button, TextField, Typography, useTheme } from '@mui/material';
import React, { useContext, useState } from 'react';
import { useSearchParams } from 'react-router';
import AuthenticationService from '../auth/AuthenticationService';
import AuthServiceContext from '../auth/AuthServiceContext';
import cfg from '../common/config';
import LoginPageBackground from './LoginPageBackground';

function LoginPage() {
  const [searchParams] = useSearchParams();
  const theme = useTheme();
  const authService = useContext<AuthenticationService>(AuthServiceContext);

  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [loginInProgress, setLoginInProgress] = useState(false);
  const [error, setError] = useState<string | undefined>();

  const clientId = searchParams.get('client_id');
  const responseType = searchParams.get('response_type');
  const scope = searchParams.get('scope');
  const state = searchParams.get('state');
  const redirectUri = searchParams.get('redirect_uri');
  const codeChallenge = searchParams.get('code_challenge');
  const codeChallengeMethod = searchParams.get('code_challenge_method');

  const isOidcLogin = responseType === 'code';

  const onSubmit = !isOidcLogin ? async (e: React.FormEvent) => {
    e.preventDefault();

    setLoginInProgress(true);

    try {
      await authService.login({ username, password });
    } catch (error) {
      if (error instanceof Error && error.message) {
        setError(error.message.includes('Unauthorized') ? 'Invalid username or password' : error.message);
      } else {
        console.error('Login error:', error);
        setError('An unknown error occurred during login');
      }
    } finally {
      setLoginInProgress(false);
    }
  } : undefined;

  const action = isOidcLogin ? `${cfg.apiUrl}/oauth/login` : undefined;
  const method = isOidcLogin ? 'POST' : undefined;

  return (
    <Box>
      <LoginPageBackground />
      <Box sx={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        minHeight: '100vh',
        padding: 2,
      }}>
        <Box
          component="form"
          method={method}
          action={action}
          onSubmit={onSubmit}
          sx={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: 2,
            width: '100%',
            maxWidth: 450
          }}
        >
          <Typography variant="h4">
            Login
          </Typography>

          <TextField
            label="Username"
            name="username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            required
            fullWidth
            sx={{ background: theme.palette.background.default }}
          />
          <TextField
            label="Password"
            name="password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            fullWidth
            sx={{ background: theme.palette.background.default }}
          />

          {clientId && <input type="hidden" name="client_id" value={clientId} />}
          {scope && <input type="hidden" name="scope" value={scope} />}
          {responseType && <input type="hidden" name="response_type" value={responseType} />}
          {state && <input type="hidden" name="state" value={state} />}
          {redirectUri && <input type="hidden" name="redirect_uri" value={redirectUri} />}
          {codeChallenge && <input type="hidden" name="code_challenge" value={codeChallenge} />}
          {codeChallengeMethod && <input type="hidden" name="code_challenge_method" value={codeChallengeMethod} />}

          {error && (
            <Alert severity="warning" sx={{ width: '100%' }}>
              {error}
            </Alert>
          )}

          <Button loading={loginInProgress} type="submit" variant="contained" sx={{ width: '100%' }}>
            Login
          </Button>
        </Box>
      </Box>
    </Box>
  );
}

export default LoginPage;
