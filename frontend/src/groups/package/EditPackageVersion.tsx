import { usePageLayout } from '@/layout/PageLayoutContext';
import { Alert, Box, Button, Chip, CircularProgress, TextField, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { createTarGzip } from 'nanotar';
import { useContext, useEffect, useState } from 'react';
import { useLazyLoadQuery } from 'react-relay/hooks';
import { Link as RouterLink, useNavigate, useParams } from 'react-router-dom';
import AuthServiceContext from '../../auth/AuthServiceContext';
import { useAppHeaderHeight } from '../../contexts/AppHeaderHeightProvider';
import { MutationError } from '../../common/error';
import { UploadPackageVersion } from './NewPackageVersion';
import { fetchPolicyFiles } from '../../packages/PackageVersionFiles';
import OPASampleInputsPanel from './OPASampleInputsPanel';
import PackageFilesEditor from './PackageFilesEditor';
import { PolicyFile, newPolicyFile } from './policyFiles';
import { EditPackageVersionQuery } from './__generated__/EditPackageVersionQuery.graphql';

const query = graphql`
    query EditPackageVersionQuery($packageId: String!, $versionId: String!) {
      pkg: node(id: $packageId) {
        ... on Package {
          id
          name
          groupPath
          # kind gates the OPA authoring aids: the sample input panel only makes sense for a policy.
          kind
        }
      }
      node(id: $versionId) {
        ... on PackageVersion {
          id
          version
          status
        }
      }
    }
`;

function EditPackageVersion() {
    const { packageId, versionId } = useParams<{ packageId: string, versionId: string }>();

    const queryData = useLazyLoadQuery<EditPackageVersionQuery>(
        query,
        { packageId: packageId as string, versionId: versionId as string },
        { fetchPolicy: 'store-and-network' }
    );
    const pkg = queryData.pkg?.name !== undefined ? queryData.pkg : null;
    const version = queryData.node?.version;

    return (
        <Box flexGrow={1} minWidth={0}>
            {pkg && version != null && <EditPackageVersionForm
                packageId={pkg.id!}
                name={pkg.name!}
                groupPath={pkg.groupPath!}
                kind={pkg.kind!}
                versionId={versionId as string}
                version={version}
            />}
            {(!pkg || version == null) && <Box display="flex" justifyContent="center" marginTop={4}>
                <Typography variant="h6" color="textSecondary">
                    pkg version not found
                </Typography>
            </Box>}
        </Box>
    );
}

interface FormProps {
    packageId: string;
    name: string;
    groupPath: string;
    kind: string;
    versionId: string;
    version: string;
}

function EditPackageVersionForm({ packageId, name, groupPath, kind, versionId, version }: FormProps) {
    // The group shell owns the page layout, so widen it rather than nesting another provider
    // (a nested provider would still be capped by the outer max width).
    usePageLayout('fullscreen');
    const navigate = useNavigate();
    const authService = useContext(AuthServiceContext);
    const { headerHeight } = useAppHeaderHeight();

    // files is null until the existing pkg has been downloaded and extracted.
    const [files, setFiles] = useState<PolicyFile[] | null>(null);
    const [showSampleInput, setShowSampleInput] = useState(false);
    const [error, setError] = useState<MutationError>();
    const [uploading, setUploading] = useState(false);

    useEffect(() => {
        let cancelled = false;
        fetchPolicyFiles(authService, versionId)
            .then(loaded => {
                if (cancelled) {
                    return;
                }
                // A version could have no files (edge case); seed with an empty file so the editor is usable.
                setFiles(loaded.length > 0 ? loaded : [newPolicyFile('policy.rego', '')]);
            })
            .catch(downloadError => {
                if (cancelled) {
                    return;
                }
                setError({ severity: 'error', message: `failed to load version files: ${downloadError.message}` });
                setFiles([newPolicyFile('policy.rego', '')]);
            });
        return () => { cancelled = true; };
    }, [authService, versionId]);

    const canSave = files != null &&
        files.some(file => file.name.trim().toLowerCase().endsWith('.rego') && file.content.trim() !== '');

    const onSave = async () => {
        if (files == null) {
            return;
        }
        setError(undefined);
        setUploading(true);
        try {
            // Build the tar.gz bytes from the edited files. Copy into a fresh ArrayBuffer-backed
            // Uint8Array so it satisfies BodyInit/BufferSource.
            const bytes = new Uint8Array(await createTarGzip(files.map(file => ({
                name: file.name,
                data: new window.TextEncoder().encode(file.content),
            }))));

            // Re-upload overwrites the existing version's pkg in place (no version bump).
            await UploadPackageVersion(authService, versionId, bytes);
            navigate(`/groups/${groupPath}/-/packages/${packageId}?tab=versions`);
        } catch (saveError: any) {
            setUploading(false);
            setError({ severity: 'warning', message: `failed to save pkg version: ${saveError.message}` });
        }
    };

    return (
        <Box sx={{ display: 'flex', flexDirection: 'column', height: `calc(100vh - ${headerHeight}px - 32px)` }}>
            <Box marginBottom={2} display="flex" justifyContent="space-between" alignItems="flex-start" gap={2}>
                <Box minWidth={0}>
                    <Box display="flex" alignItems="center" gap={1}>
                        <Typography variant="h6">{name}</Typography>
                        <Chip variant="outlined" size="small" label={`editing ${version}`} />
                    </Box>
                    <Typography variant="caption" color="textSecondary">{groupPath}</Typography>
                </Box>
                {kind === 'OPA_POLICY' && (
                    <Button
                        variant="outlined"
                        color="secondary"
                        onClick={() => setShowSampleInput(prev => !prev)}
                        sx={{ flexShrink: 0 }}
                    >
                        {showSampleInput ? 'Hide sample inputs' : 'Show sample inputs'}
                    </Button>
                )}
            </Box>

            {error && <Alert sx={{ marginBottom: 2 }} severity={error.severity}>{error.message}</Alert>}

            <TextField
                label="Version"
                size="small"
                fullWidth
                margin="normal"
                value={version}
                disabled
                helperText="Editing a version's pkg does not change its version number."
            />

            <Box sx={{ flexGrow: 1, minHeight: 0, marginTop: 2, display: 'flex', gap: 2 }}>
                <Box sx={{ flex: 1, minWidth: 0 }}>
                    {files == null
                        ? <Box display="flex" justifyContent="center" padding={4}><CircularProgress /></Box>
                        : <PackageFilesEditor files={files} onChange={setFiles} />}
                </Box>

                {kind === 'OPA_POLICY' && showSampleInput && <OPASampleInputsPanel />}
            </Box>

            <Box marginTop={2}>
                <Button
                    sx={{ marginRight: 2 }}
                    loading={uploading}
                    disabled={!canSave}
                    variant="outlined"
                    color="primary"
                    onClick={onSave}
                >
                    Save
                </Button>
                <Button component={RouterLink} color="inherit" to={`/groups/${groupPath}/-/packages/${packageId}?tab=versions`}>Cancel</Button>
            </Box>
        </Box>
    );
}

export default EditPackageVersion;
