import SmartToyIcon from '@mui/icons-material/SmartToy';
import { Alert, AlertTitle, Box, Button, Chip, Paper, TextField, Tooltip, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import { useMemo, useState } from 'react';
import { useLazyLoadQuery, useMutation } from 'react-relay/hooks';
import { useNavigate, useParams } from 'react-router-dom';
import NamespaceBreadcrumbs from '../NamespaceBreadcrumbs';
import Gravatar from '../../common/Gravatar';
import Timestamp from '../../common/Timestamp';
import ConfirmationDialog from '../../common/ConfirmationDialog';
import { ProviderMirrorVersionDetailsQuery } from './__generated__/ProviderMirrorVersionDetailsQuery.graphql';
import { ProviderMirrorVersionDetailsDeletePlatformMutation } from './__generated__/ProviderMirrorVersionDetailsDeletePlatformMutation.graphql';
import { ProviderMirrorVersionDetailsDeleteMutation } from './__generated__/ProviderMirrorVersionDetailsDeleteMutation.graphql';

interface Props {
    namespacePath: string
}

function ProviderMirrorVersionDetails({ namespacePath }: Props) {
    const { mirrorId } = useParams();
    const navigate = useNavigate();
    const theme = useTheme();
    const { enqueueSnackbar } = useSnackbar();
    const [platformToDelete, setPlatformToDelete] = useState<{ id: string, os: string, arch: string } | null>(null);
    const [showDeleteVersion, setShowDeleteVersion] = useState(false);
    const [confirmInput, setConfirmInput] = useState('');

    const data = useLazyLoadQuery<ProviderMirrorVersionDetailsQuery>(graphql`
        query ProviderMirrorVersionDetailsQuery($id: String!) {
            node(id: $id) {
                ... on TerraformProviderVersionMirror {
                    id
                    metadata {
                        createdAt
                    }
                    version
                    createdBy
                    providerAddress
                    groupPath
                    platformMirrors {
                        id
                        os
                        arch
                    }
                }
            }
        }
    `, { id: mirrorId as string }, { fetchPolicy: 'store-and-network' });

    const [commitDelete, isDeleting] = useMutation<ProviderMirrorVersionDetailsDeletePlatformMutation>(graphql`
        mutation ProviderMirrorVersionDetailsDeletePlatformMutation($input: DeleteTerraformProviderPlatformMirrorInput!) {
            deleteTerraformProviderPlatformMirror(input: $input) {
                problems {
                    message
                }
            }
        }
    `);

    const [commitDeleteVersion, isDeletingVersion] = useMutation<ProviderMirrorVersionDetailsDeleteMutation>(graphql`
        mutation ProviderMirrorVersionDetailsDeleteMutation($input: DeleteTerraformProviderVersionMirrorInput!) {
            deleteTerraformProviderVersionMirror(input: $input) {
                problems {
                    message
                }
            }
        }
    `);

    const mirror = data.node as any;

    if (!mirror) {
        return (
            <Box display="flex" justifyContent="center" marginTop={4}>
                <Typography color="textSecondary">Provider mirror with ID {mirrorId} not found</Typography>
            </Box>
        );
    }

    const isInherited = mirror.groupPath !== namespacePath;
    const platforms = useMemo(() =>
        [...(mirror.platformMirrors ?? [])].sort((a: any, b: any) =>
            `${a.os}/${a.arch}`.localeCompare(`${b.os}/${b.arch}`)
        ), [mirror.platformMirrors]);

    const onDeletePlatform = (confirm?: boolean) => {
        if (!confirm || !platformToDelete) {
            setPlatformToDelete(null);
            return;
        }
        commitDelete({
            variables: { input: { id: platformToDelete.id } },
            onCompleted: (response) => {
                setPlatformToDelete(null);
                if (response.deleteTerraformProviderPlatformMirror?.problems?.length) {
                    enqueueSnackbar(response.deleteTerraformProviderPlatformMirror.problems[0].message, { variant: 'warning' });
                } else {
                    enqueueSnackbar('Platform mirror deleted', { variant: 'success' });
                }
            },
            onError: (err) => {
                setPlatformToDelete(null);
                enqueueSnackbar(`Unexpected error: ${err.message}`, { variant: 'error' });
            },
            updater: (store) => {
                const mirrorRecord = store.get(mirror.id);
                const platforms = mirrorRecord?.getLinkedRecords('platformMirrors');
                if (mirrorRecord && platforms) {
                    mirrorRecord.setLinkedRecords(
                        platforms.filter(p => p?.getDataID() !== platformToDelete.id),
                        'platformMirrors'
                    );
                }
            }
        });
    };

    const onDeleteVersionDialogClosed = (confirm?: boolean) => {
        if (confirm && mirror?.id) {
            commitDeleteVersion({
                variables: { input: { id: mirror.id, force: true } },
                onCompleted: (response) => {
                    setShowDeleteVersion(false);
                    if (response.deleteTerraformProviderVersionMirror?.problems?.length) {
                        enqueueSnackbar(response.deleteTerraformProviderVersionMirror.problems[0].message, { variant: 'warning' });
                    } else {
                        enqueueSnackbar('Provider mirror deleted', { variant: 'success' });
                        navigate('..');
                    }
                },
                onError: (err) => {
                    setShowDeleteVersion(false);
                    enqueueSnackbar(`Unexpected error: ${err.message}`, { variant: 'error' });
                }
            });
        } else {
            setShowDeleteVersion(false);
        }
    };

    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={namespacePath}
                childRoutes={[
                    { title: 'provider_mirror', path: 'provider_mirror' },
                    { title: mirror.providerAddress, path: mirrorId as string }
                ]}
            />

            <Box mb={3} display="flex" justifyContent="space-between" alignItems="flex-start">
                <Box>
                    <Box display="flex" alignItems="center" gap={1} mb={1}>
                        <Typography variant="h5">{mirror.providerAddress}</Typography>
                        <Chip label={`v${mirror.version}`} size="small" />
                    </Box>
                    <Box display="flex" alignItems="center">
                        <Typography variant="body2" color="textSecondary">
                            Cached <Timestamp component="span" timestamp={mirror.metadata?.createdAt} /> by
                        </Typography>
                        {mirror.createdBy.startsWith('trn:') ? (
                            <Tooltip title={mirror.createdBy}>
                                <SmartToyIcon sx={{ ml: 0.5, width: 20, height: 20, color: 'text.secondary' }} />
                            </Tooltip>
                        ) : (
                            <Tooltip title={mirror.createdBy}>
                                <Box><Gravatar width={20} height={20} sx={{ ml: 0.5 }} email={mirror.createdBy} /></Box>
                            </Tooltip>
                        )}
                    </Box>
                </Box>
                {!isInherited && (
                    <Button variant="outlined" color="error" onClick={() => setShowDeleteVersion(true)}>
                        Delete Mirror
                    </Button>
                )}
            </Box>

            <Paper sx={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0, border: `1px solid ${theme.palette.divider}` }}>
                <Box padding={2}>
                    <Typography variant="subtitle1">{platforms.length} platform{platforms.length === 1 ? '' : 's'}</Typography>
                </Box>
            </Paper>
            {platforms.length === 0 ? (
                <Box sx={{ border: `1px solid ${theme.palette.divider}`, borderTop: 0, borderBottomLeftRadius: 4, borderBottomRightRadius: 4, p: 2 }}>
                    <Typography color="textSecondary">No platforms mirrored yet.</Typography>
                </Box>
            ) : (
                <Box sx={{ border: `1px solid ${theme.palette.divider}`, borderTop: 0, borderBottomLeftRadius: 4, borderBottomRightRadius: 4, p: 2 }}>
                    <Box display="flex" flexWrap="wrap" gap={1}>
                        {platforms.map((platform) => (
                            <Chip
                                key={platform.id}
                                label={`${platform.os}/${platform.arch}`}
                                onDelete={!isInherited ? () => setPlatformToDelete(platform) : undefined}
                            />
                        ))}
                    </Box>
                </Box>
            )}

            {platformToDelete && (
                <ConfirmationDialog
                    title="Delete Platform Mirror"
                    confirmLabel="Delete"
                    confirmInProgress={isDeleting}
                    onConfirm={() => onDeletePlatform(true)}
                    onClose={() => onDeletePlatform()}
                >
                    Are you sure you want to delete the cached package for <strong>{platformToDelete.os}/{platformToDelete.arch}</strong>?
                </ConfirmationDialog>
            )}

            {showDeleteVersion && (
                <ConfirmationDialog
                    title="Delete Provider Mirror"
                    maxWidth="sm"
                    confirmLabel="Delete"
                    confirmDisabled={confirmInput !== mirror.providerAddress}
                    confirmInProgress={isDeletingVersion}
                    onConfirm={() => onDeleteVersionDialogClosed(true)}
                    onClose={() => onDeleteVersionDialogClosed()}
                >
                    <Alert severity="warning">
                        <AlertTitle>Warning</AlertTitle>
                        Deleting this provider mirror will remove all cached packages for <strong>{mirror.providerAddress}/{mirror.version}</strong>. Runs using this provider version will need to download it from the upstream registry. This action <strong><ins>cannot be undone</ins></strong>.
                    </Alert>
                    <Typography variant="subtitle2" sx={{ mt: 2, mb: 1 }}>Type <strong>{mirror.providerAddress}</strong> to confirm:</Typography>
                    <TextField
                        autoComplete="off"
                        fullWidth
                        size="small"
                        placeholder={mirror.providerAddress}
                        value={confirmInput}
                        onChange={(e) => setConfirmInput(e.target.value)}
                    />
                </ConfirmationDialog>
            )}
        </Box>
    );
}

export default ProviderMirrorVersionDetails;
