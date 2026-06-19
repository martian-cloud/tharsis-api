import VisibilityIcon from '@mui/icons-material/Visibility';
import VisibilityOffIcon from '@mui/icons-material/VisibilityOff';
import {
    Box,
    IconButton,
    List,
    Table,
    TableBody,
    TableCell,
    TableContainer,
    TableHead,
    TableRow,
    ToggleButton,
    ToggleButtonGroup,
    Tooltip,
    Typography,
    useMediaQuery,
    useTheme,
} from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { memo, useMemo, useState } from 'react';
import { useLazyLoadQuery } from 'react-relay/hooks';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { atomDark as prismTheme } from 'react-syntax-highlighter/dist/esm/styles/prism';
import CopyButton from '../../common/CopyButton';
import DataTableCell from '../../common/DataTableCell';
import SearchInput from '../../common/SearchInput';
import { AdminAreaConfigurationSettingsQuery } from './__generated__/AdminAreaConfigurationSettingsQuery.graphql';

const EMPTY = '—';

const query = graphql`
    query AdminAreaConfigurationSettingsQuery {
        config {
            serverPort
            tharsisApiUrl
            tharsisUiUrl
            tharsisSupportUrl
            serviceDiscoveryHost
            corsAllowedOrigins
            tlsEnabled
            httpRateLimit
            jwtIssuerUrl
            oidcInternalIdentityProviderClientID
            cliLoginOIDCClientID
            cliLoginOIDCScopes
            oauthProviders {
                issuerUrl
                clientId
                usernameClaim
                scope
            }
            userSessionAccessTokenExpirationMinutes
            userSessionRefreshTokenExpirationMinutes
            userSessionMaxSessionsPerUser
            maxGraphQlComplexity
            moduleRegistryMaxUploadSize
            asyncTaskTimeout
            vcsRepositorySizeLimit
            serviceAccountClientSecretMaxExpirationDays
            terraformCliVersionConstraint
            workspaceAssessmentIntervalHours
            workspaceAssessmentRunLimit
            asymmetricSigningKeyRotationPeriodDays
            asymmetricSigningKeyDecommissionPeriodDays
            aiEnabled
            disableSensitiveVariableFeature
            emailFooter
            objectStorePluginType
            rateLimitStorePluginType
            jwsProviderPluginType
            secretManagerPluginType
            emailClientPluginType
            adminLogTailStorePluginType
            objectStorePluginData {
                key
                value
            }
            rateLimitStorePluginData {
                key
                value
            }
            jwsProviderPluginData {
                key
                value
            }
            secretManagerPluginData {
                key
                value
            }
            emailClientPluginData {
                key
                value
            }
            adminLogTailStorePluginData {
                key
                value
            }
            dbHost
            dbName
            dbSslMode
            dbPort
            dbMaxConnections
            dbAutoMigrateEnabled
            tlsCertFile
            tlsKeyFile
            adminUserEmail
            otelTraceEnabled
            otelTraceType
            otelTraceCollectorHost
            otelTraceCollectorPort
            federatedRegistryTrustPolicies {
                issuerUrl
                subject
                audience
                groupGlobPatterns
            }
            internalRunners {
                name
                jobDispatcherType
                jobDispatcherData {
                    key
                    value
                }
            }
            mcpServerConfig {
                enabledToolsets
                enabledTools
                readOnly
            }
            sensitiveFields
        }
    }
`;

// describeValue renders a config value for display. Scalars and string lists are inline
// text; nested objects/arrays become a JSON string shown in a syntax-highlighted block.
function describeValue(value: unknown): { text: string, code: boolean } {
    if (value === null || value === undefined || value === '') {
        return { text: EMPTY, code: false };
    }

    if (Array.isArray(value)) {
        if (value.length === 0) {
            return { text: EMPTY, code: false };
        }

        if (value.every((item) => typeof item === 'string')) {
            return { text: value.join(', '), code: false };
        }

        return { text: JSON.stringify(value, null, 2), code: true };
    }

    if (typeof value === 'object') {
        return { text: JSON.stringify(value, null, 2), code: true };
    }

    return { text: String(value), code: false };
}

function JsonBlock({ json }: { json: string }) {
    return (
        <SyntaxHighlighter
            language="json"
            style={prismTheme}
            wrapLongLines
            customStyle={{ margin: 0, borderRadius: 4, fontSize: '0.8rem', whiteSpace: 'pre-wrap', overflowWrap: 'anywhere' }}
            codeTagProps={{ style: { whiteSpace: 'pre-wrap', overflowWrap: 'anywhere', wordBreak: 'break-word' } }}
        >
            {json}
        </SyntaxHighlighter>
    );
}

const ConfigRow = memo(function ConfigRow({ field, text, code, sensitive, mobile }: { field: string, text: string, code: boolean, sensitive: boolean, mobile: boolean }) {
    const [revealed, setRevealed] = useState(false);
    const isSecret = sensitive && text !== EMPTY;
    const masked = isSecret && !revealed;

    const actions = (
        <>
            {isSecret && (
                <Tooltip title={revealed ? 'Hide' : 'Reveal'} placement="top">
                    <IconButton size="small" onClick={() => setRevealed((prev) => !prev)}>
                        {revealed ? <VisibilityOffIcon sx={{ width: 16, height: 16 }} /> : <VisibilityIcon sx={{ width: 16, height: 16 }} />}
                    </IconButton>
                </Tooltip>
            )}
            {!masked && text !== EMPTY && <CopyButton data={text} toolTip="Copy value" />}
        </>
    );

    // For revealed code blocks the actions are pinned to the top-right so a tall/wide
    // value can't push them off-screen; inline values keep the actions alongside.
    const value = code && !masked ? (
        <Box sx={{ position: 'relative' }}>
            <Box sx={{ position: 'absolute', top: 4, right: 4, zIndex: 1, display: 'flex', gap: 0.5 }}>{actions}</Box>
            <JsonBlock json={text} />
        </Box>
    ) : (
        <Box display="flex" alignItems="center" gap={0.5}>
            {masked
                ? <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>••••••••</Typography>
                : <Typography variant="body2" sx={{ fontFamily: 'monospace', whiteSpace: 'pre-wrap', overflowWrap: 'anywhere' }}>{text}</Typography>}
            {actions}
        </Box>
    );

    if (mobile) {
        return (
            <Box component="li" sx={{ listStyle: 'none', px: 2, py: 1.5, borderBottom: 1, borderColor: 'divider', '&:last-of-type': { borderBottom: 0 } }}>
                <Typography variant="caption" color="textSecondary" sx={{ display: 'block', fontFamily: 'monospace', overflowWrap: 'anywhere' }}>{field}</Typography>
                <Box sx={{ mt: 0.5 }}>{value}</Box>
            </Box>
        );
    }

    return (
        <TableRow hover sx={{ '&:last-child td, &:last-child th': { border: 0 }, verticalAlign: 'top' }}>
            <DataTableCell sx={{ whiteSpace: 'nowrap' }}>{field}</DataTableCell>
            <TableCell sx={{ width: '100%' }}>{value}</TableCell>
        </TableRow>
    );
});

function ConfigJsonView({ json }: { json: string }) {
    return (
        <Box sx={{ position: 'relative' }}>
            <Box sx={{ position: 'absolute', top: 4, right: 4, zIndex: 1 }}>
                <CopyButton data={json} toolTip="Copy configuration JSON" />
            </Box>
            <JsonBlock json={json} />
        </Box>
    );
}

function AdminAreaConfigurationSettings() {
    const theme = useTheme();
    const mobile = useMediaQuery(theme.breakpoints.down('sm'));
    const data = useLazyLoadQuery<AdminAreaConfigurationSettingsQuery>(query, {}, { fetchPolicy: 'store-and-network' });
    const config = data.config;

    const [viewMode, setViewMode] = useState<'table' | 'json'>('table');
    const [search, setSearch] = useState('');

    // Every config field except the sensitiveFields metadata list.
    const entries = useMemo(() => Object.entries(config).filter(([field]) => field !== 'sensitiveFields'), [config]);

    const rows = useMemo(() => {
        const sensitiveFields = new Set(config.sensitiveFields);
        return entries
            .map(([field, value]) => ({ field, sensitive: sensitiveFields.has(field.toLowerCase()), ...describeValue(value) }))
            .sort((a, b) => a.field.localeCompare(b.field));
    }, [entries, config.sensitiveFields]);

    const json = useMemo(() => JSON.stringify(Object.fromEntries(entries), null, 2), [entries]);

    const normalizedSearch = search.trim().toLowerCase();
    const filteredRows = useMemo(
        () => normalizedSearch
            ? rows.filter((row) => row.field.toLowerCase().includes(normalizedSearch) || row.text.toLowerCase().includes(normalizedSearch))
            : rows,
        [rows, normalizedSearch]
    );

    const rowElements = filteredRows.map((row) => (
        <ConfigRow key={row.field} field={row.field} text={row.text} code={row.code} sensitive={row.sensitive} mobile={mobile} />
    ));

    return (
        <Box>
            <Typography variant="h5" gutterBottom>API Configuration</Typography>
            <Typography variant="body2" sx={{ mb: 3 }}>
                Read-only view of the running API server configuration.
            </Typography>

            <Box display="flex" alignItems="center" gap={2} sx={{ mb: 2 }}>
                {viewMode === 'table' && (
                    <SearchInput
                        fullWidth
                        placeholder="filter configuration"
                        value={search}
                        onChange={(event) => setSearch(event.target.value)}
                    />
                )}
                <ToggleButtonGroup
                    size="small"
                    color="primary"
                    value={viewMode}
                    exclusive
                    onChange={(_event, value) => value && setViewMode(value)}
                    sx={{ ml: viewMode === 'table' ? 0 : 'auto' }}
                >
                    <ToggleButton value="table">Table</ToggleButton>
                    <ToggleButton value="json">JSON</ToggleButton>
                </ToggleButtonGroup>
            </Box>

            {viewMode === 'json' && <ConfigJsonView json={json} />}

            {viewMode === 'table' && filteredRows.length === 0 && (
                <Typography color="textSecondary" align="center" sx={{ p: 2 }}>
                    No configuration matching <strong>{search}</strong>
                </Typography>
            )}

            {viewMode === 'table' && filteredRows.length > 0 && (
                mobile ? (
                    <List disablePadding>{rowElements}</List>
                ) : (
                    <TableContainer>
                        <Table>
                            <TableHead>
                                <TableRow>
                                    <TableCell><Typography color="textSecondary">Field</Typography></TableCell>
                                    <TableCell><Typography color="textSecondary">Value</Typography></TableCell>
                                </TableRow>
                            </TableHead>
                            <TableBody>{rowElements}</TableBody>
                        </Table>
                    </TableContainer>
                )
            )}
        </Box>
    );
}

export default AdminAreaConfigurationSettings;
