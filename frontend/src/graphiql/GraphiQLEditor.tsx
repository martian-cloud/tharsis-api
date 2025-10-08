import { createGraphiQLFetcher, Fetcher } from '@graphiql/toolkit';
import { Box } from '@mui/material';
import GraphiQL from 'graphiql';
import 'graphiql/graphiql.css';
import { useContext } from 'react';
import AuthenticationService from '../auth/AuthenticationService';
import AuthServiceContext from '../auth/AuthServiceContext';
import cfg from '../common/config';
import { useAppHeaderHeight } from '../contexts/AppHeaderHeightProvider';

const fetcher = (authService: AuthenticationService): Fetcher => createGraphiQLFetcher({
    url: `${cfg.apiUrl}/graphql`,
    fetch: (input: URL | RequestInfo, config?: RequestInit | undefined) => {
        return authService.fetchWithAuth(input, config) // nosemgrep: nodejs_scan.javascript-ssrf-rule-node_ssrf
    }
});

function GraphiQLEditor() {
    const authService = useContext<AuthenticationService>(AuthServiceContext)
    const { headerHeight } = useAppHeaderHeight();

    const f = fetcher(authService);

    return (
        <Box sx={{
            position: 'absolute',
            top: `${headerHeight}px`,
            left: 0,
            width: '100%',
            height: `calc(100% - ${headerHeight}px)`,
            display: 'flex',
            flexDirection: 'column'
        }}>
            <GraphiQL
                fetcher={f}
            />
        </Box>
    );
}

export default GraphiQLEditor
