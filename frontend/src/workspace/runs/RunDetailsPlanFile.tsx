import { CircularProgress, Paper, Typography } from '@mui/material';
import Box from '@mui/material/Box';
import { useContext, useEffect, useState } from 'react';
import SyntaxHighlighter from 'react-syntax-highlighter';
import { a11yDark } from 'react-syntax-highlighter/dist/esm/styles/hljs';
import AuthenticationService from '../../auth/AuthenticationService';
import AuthServiceContext from '../../auth/AuthServiceContext';
import cfg from '../../common/config';

// MaxPlanFileSize caps how large a plan JSON file is rendered inline to avoid freezing the browser
// on very large plans (mirrors the diff viewer's MaxDiffSize guard).
export const MaxPlanFileSize = 1024 * 1024; // 1MB

interface Props {
    planId: string
}

// RunDetailsPlanFile fetches the run's plan JSON from the REST endpoint and renders it as
// formatted JSON. The plan JSON is not exposed through GraphQL, so it is downloaded directly
// (authenticated via the shared session) rather than through Relay.
function RunDetailsPlanFile({ planId }: Props) {
    const authService = useContext<AuthenticationService>(AuthServiceContext);
    const [content, setContent] = useState<string | null>(null);
    const [tooLarge, setTooLarge] = useState<boolean>(false);
    const [loading, setLoading] = useState<boolean>(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        let cancelled = false;

        const loadPlanFile = async () => {
            setLoading(true);
            setError(null);
            setTooLarge(false);
            setContent(null);

            try {
                const response = await authService.fetchWithAuth( // nosemgrep: nodejs_scan.javascript-ssrf-rule-node_ssrf
                    `${cfg.apiUrl}/tfe/v2/plans/${planId}/json-output`,
                    { method: 'GET' }
                );

                if (!response.ok) {
                    if (response.status === 403) {
                        throw new Error('You do not have permission to view the plan file for this run');
                    }
                    throw new Error(`Failed to load plan file (status ${response.status})`);
                }

                const text = await response.text();
                if (cancelled) {
                    return;
                }

                if (text.length > MaxPlanFileSize) {
                    setTooLarge(true);
                    return;
                }

                // The endpoint returns application/json; pretty-print it when possible and fall
                // back to the raw payload otherwise.
                let formatted = text;
                try {
                    formatted = JSON.stringify(JSON.parse(text), null, 2);
                } catch {
                    // Not valid JSON, show the raw payload as-is.
                }
                setContent(formatted);
            } catch (err: any) {
                if (!cancelled) {
                    setError(err?.message ?? 'Failed to load plan file');
                }
            } finally {
                if (!cancelled) {
                    setLoading(false);
                }
            }
        };

        loadPlanFile();

        return () => {
            cancelled = true;
        };
    }, [planId, authService]);

    if (loading) {
        return (
            <Box sx={{ minHeight: 120, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <CircularProgress />
            </Box>
        );
    }

    if (error) {
        return (
            <Paper variant="outlined" sx={{ minHeight: 100, display: 'flex', flexDirection: 'column', justifyContent: 'center' }}>
                <Typography color="textSecondary" align="center">{error}</Typography>
            </Paper>
        );
    }

    if (tooLarge) {
        return (
            <Paper variant="outlined" sx={{ minHeight: 100, display: 'flex', flexDirection: 'column', justifyContent: 'center' }}>
                <Typography color="textSecondary" align="center">
                    Plan file is too large to display. The file exceeds the maximum limit of {MaxPlanFileSize} bytes
                </Typography>
            </Paper>
        );
    }

    if (!content) {
        return (
            <Paper variant="outlined" sx={{ minHeight: 100, display: 'flex', flexDirection: 'column', justifyContent: 'center' }}>
                <Typography color="textSecondary" align="center">The plan file is empty</Typography>
            </Paper>
        );
    }

    return (
        <Box sx={theme => ({ fontSize: theme.typography.code.fontSize, overflowX: 'auto' })}>
            <SyntaxHighlighter language="json" style={a11yDark}>
                {content}
            </SyntaxHighlighter>
        </Box>
    );
}

export default RunDetailsPlanFile;
