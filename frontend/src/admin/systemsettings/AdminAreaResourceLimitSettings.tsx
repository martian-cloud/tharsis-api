import EditIcon from '@mui/icons-material/EditOutlined';
import {
    Alert,
    Box,
    Button,
    List,
    Stack,
    Table,
    TableBody,
    TableCell,
    TableContainer,
    TableHead,
    TableRow,
    TextField,
    Tooltip,
    Typography,
    useMediaQuery,
    useTheme,
} from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import { useMemo, useState } from 'react';
import { useLazyLoadQuery, useMutation } from 'react-relay/hooks';
import DataTableCell from '../../common/DataTableCell';
import { MutationError } from '../../common/error';
import SearchInput from '../../common/SearchInput';
import Timestamp from '../../common/Timestamp';
import { AdminAreaResourceLimitSettingsQuery } from './__generated__/AdminAreaResourceLimitSettingsQuery.graphql';
import { AdminAreaResourceLimitSettingsUpdateMutation } from './__generated__/AdminAreaResourceLimitSettingsUpdateMutation.graphql';

const query = graphql`
    query AdminAreaResourceLimitSettingsQuery {
        resourceLimits {
            id
            name
            value
            metadata {
                version
                updatedAt
            }
        }
    }
`;

const updateMutation = graphql`
    mutation AdminAreaResourceLimitSettingsUpdateMutation($input: UpdateResourceLimitInput!) {
        updateResourceLimit(input: $input) {
            resourceLimit {
                id
                name
                value
                metadata {
                    version
                    updatedAt
                }
            }
            problems {
                message
                field
            }
        }
    }
`;

interface LimitRowData {
    id: string;
    name: string;
    display: string;
    value: number;
    version: string;
    updatedAt: string;
}

// isValidValue accepts only non-negative integers; limits are counts and cannot be negative.
function isValidValue(draft: string): boolean {
    return /^\d+$/.test(draft.trim());
}

// displayName drops the shared "ResourceLimit" prefix, splits the PascalCase name into spaced
// words (keeping acronyms like GPG/VCS intact), and renders "Per" as "/" for readability.
function displayName(name: string): string {
    return name
        .replace(/^ResourceLimit/, '')
        .replace(/([A-Z]+)([A-Z][a-z])/g, '$1 $2')
        .replace(/([a-z\d])([A-Z])/g, '$1 $2')
        .replace(/\bPer\b/g, '/');
}

interface LimitRowProps {
    limit: LimitRowData;
    editing: boolean;
    saving: boolean;
    mobile: boolean;
    onEdit: () => void;
    onCancel: () => void;
    onSave: (value: number) => void;
}

function LimitRow({ limit, editing, saving, mobile, onEdit, onCancel, onSave }: LimitRowProps) {
    const [draft, setDraft] = useState(String(limit.value));

    const startEdit = () => {
        setDraft(String(limit.value));
        onEdit();
    };

    const valid = isValidValue(draft);
    const changed = valid && Number(draft.trim()) !== limit.value;

    const editField = (
        <TextField
            size="small"
            type="number"
            value={draft}
            autoFocus
            error={!valid}
            disabled={saving}
            fullWidth={mobile}
            onChange={(event) => setDraft(event.target.value)}
            slotProps={{ htmlInput: { min: 0, 'aria-label': `Value for ${limit.name}` } }}
            sx={mobile ? undefined : { width: 140 }}
        />
    );

    const saveCancel = (
        <Stack direction="row" spacing={1} justifyContent="flex-end">
            <Button size="small" variant="outlined" color="primary" loading={saving} disabled={!changed} onClick={() => onSave(Number(draft.trim()))}>Save</Button>
            <Button size="small" color="inherit" disabled={saving} onClick={onCancel}>Cancel</Button>
        </Stack>
    );

    const editButton = (
        <Tooltip title="Edit" placement="top">
            <Button onClick={startEdit} sx={{ minWidth: 40, padding: '2px' }} size="small" color="info" variant="outlined">
                <EditIcon />
            </Button>
        </Tooltip>
    );

    if (mobile) {
        return (
            <Box component="li" sx={{ listStyle: 'none', px: 2, py: 1.5, borderBottom: 1, borderColor: 'divider', '&:last-of-type': { borderBottom: 0 } }}>
                <Box display="flex" alignItems="center" gap={1}>
                    <Typography variant="body2" sx={{ flexGrow: 1, overflowWrap: 'anywhere' }}>{limit.display}</Typography>
                    {!editing && editButton}
                </Box>
                {editing ? (
                    <Box sx={{ mt: 1, display: 'flex', flexDirection: 'column', gap: 1 }}>
                        {editField}
                        {saveCancel}
                    </Box>
                ) : (
                    <Box sx={{ mt: 0.5, display: 'flex', alignItems: 'center', gap: 0.5 }}>
                        <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>{limit.value}</Typography>
                        <Box sx={{ flexGrow: 1 }} />
                        <Box sx={{ display: 'flex', alignItems: 'baseline', gap: 0.5 }}>
                            <Typography variant="caption" color="textSecondary">Updated</Typography>
                            <Timestamp timestamp={limit.updatedAt} variant="caption" color="textSecondary" />
                        </Box>
                    </Box>
                )}
            </Box>
        );
    }

    return (
        <TableRow hover sx={{ '&:last-child td, &:last-child th': { border: 0 } }}>
            <TableCell>{limit.display}</TableCell>
            {editing ? <TableCell>{editField}</TableCell> : <DataTableCell>{limit.value}</DataTableCell>}
            <TableCell>
                <Timestamp timestamp={limit.updatedAt} variant="body2" color="textSecondary" />
            </TableCell>
            <TableCell align="right">
                {editing ? saveCancel : editButton}
            </TableCell>
        </TableRow>
    );
}

function AdminAreaResourceLimitSettings() {
    const theme = useTheme();
    const mobile = useMediaQuery(theme.breakpoints.down('sm'));
    const { enqueueSnackbar } = useSnackbar();
    const data = useLazyLoadQuery<AdminAreaResourceLimitSettingsQuery>(query, {}, { fetchPolicy: 'store-and-network' });

    const [commit, isInFlight] = useMutation<AdminAreaResourceLimitSettingsUpdateMutation>(updateMutation);
    const [error, setError] = useState<MutationError>();
    const [editingName, setEditingName] = useState<string>();
    const [savingName, setSavingName] = useState<string>();
    const [search, setSearch] = useState('');

    const limits = useMemo<LimitRowData[]>(
        () => data.resourceLimits
            .map((limit) => ({ id: limit.id, name: limit.name, display: displayName(limit.name), value: limit.value, version: limit.metadata.version, updatedAt: limit.metadata.updatedAt }))
            .sort((a, b) => a.display.localeCompare(b.display)),
        [data.resourceLimits]
    );

    const normalizedSearch = search.trim().toLowerCase();
    const filtered = useMemo(
        () => normalizedSearch ? limits.filter((limit) => limit.display.toLowerCase().includes(normalizedSearch)) : limits,
        [limits, normalizedSearch]
    );

    const onSave = (limit: LimitRowData, value: number) => {
        setError(undefined);
        setSavingName(limit.name);

        commit({
            variables: { input: { name: limit.name, value, metadata: { version: limit.version } } },
            onCompleted: (response) => {
                setSavingName(undefined);

                const problems = response.updateResourceLimit.problems;
                if (problems.length) {
                    setError({ severity: 'warning', message: problems.map((problem) => problem.message).join(', ') });
                    return;
                }

                setEditingName(undefined);
                enqueueSnackbar('Resource limit updated', { variant: 'success' });
            },
            onError: (commitError: Error) => {
                setSavingName(undefined);
                setError({ severity: 'error', message: `Unexpected error occurred: ${commitError.message}` });
            }
        });
    };

    const rows = filtered.map((limit) => (
        <LimitRow
            key={limit.id}
            limit={limit}
            mobile={mobile}
            editing={editingName === limit.name}
            saving={savingName === limit.name && isInFlight}
            onEdit={() => { setError(undefined); setEditingName(limit.name); }}
            onCancel={() => setEditingName(undefined)}
            onSave={(value) => onSave(limit, value)}
        />
    ));

    return (
        <Box>
            <Typography variant="h5" gutterBottom>Resource Limits</Typography>
            <Typography variant="body2" sx={{ mb: 3 }}>
                System-wide limits enforced across all groups and workspaces.
            </Typography>

            {error && <Alert severity={error.severity} sx={{ mb: 2 }}>{error.message}</Alert>}

            <Box sx={{ mb: 2 }}>
                <SearchInput
                    fullWidth
                    placeholder="filter resource limits"
                    value={search}
                    onChange={(event) => { setSearch(event.target.value); setEditingName(undefined); }}
                />
            </Box>

            {filtered.length === 0 ? (
                <Typography color="textSecondary" align="center" sx={{ p: 2 }}>
                    No resource limits matching <strong>{search}</strong>
                </Typography>
            ) : mobile ? (
                <List disablePadding>{rows}</List>
            ) : (
                <TableContainer>
                    <Table>
                        <TableHead>
                            <TableRow>
                                <TableCell><Typography color="textSecondary">Name</Typography></TableCell>
                                <TableCell><Typography color="textSecondary">Value</Typography></TableCell>
                                <TableCell><Typography color="textSecondary">Last Updated</Typography></TableCell>
                                <TableCell></TableCell>
                            </TableRow>
                        </TableHead>
                        <TableBody>{rows}</TableBody>
                    </Table>
                </TableContainer>
            )}
        </Box>
    );
}

export default AdminAreaResourceLimitSettings;
