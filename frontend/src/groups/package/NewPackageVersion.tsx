import OpenInNewIcon from '@mui/icons-material/OpenInNew';
import { usePageLayout } from '@/layout/PageLayoutContext';
import { Alert, Box, Button, Chip, CircularProgress, Link, TextField, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { createTarGzip } from 'nanotar';
import { useContext, useEffect, useState } from 'react';
import { useLazyLoadQuery, useMutation } from 'react-relay/hooks';
import { Link as RouterLink, useNavigate, useParams } from 'react-router-dom';
import AuthServiceContext from '../../auth/AuthServiceContext';
import AuthenticationService from '../../auth/AuthenticationService';
import cfg from '../../common/config';
import { MutationError } from '../../common/error';
import { useAppHeaderHeight } from '../../contexts/AppHeaderHeightProvider';
import { fetchPolicyFiles } from '../../packages/PackageVersionFiles';
import OPASampleInputsPanel from './OPASampleInputsPanel';
import PackageFilesEditor from './PackageFilesEditor';
import { PolicyFile, newPolicyFile } from './policyFiles';
import { NewPackageVersionCreateMutation } from './__generated__/NewPackageVersionCreateMutation.graphql';
import { NewPackageVersionQuery } from './__generated__/NewPackageVersionQuery.graphql';
import { SAMPLE_OPA_POLICY } from './opaSamples';

const query = graphql`
    query NewPackageVersionQuery($id: String!) {
      node(id: $id) {
        ... on Package {
          id
          name
          groupPath
          kind
          # The latest version seeds the editor, so a new version starts from what is published rather
          # than from the starter policy. Its status decides whether the package can be downloaded.
          latestVersion {
            id
            status
          }
        }
      }
    }
`;

const createMutation = graphql`
    mutation NewPackageVersionCreateMutation($input: CreatePackageVersionInput!) {
        createPackageVersion(input: $input) {
            packageVersion {
                id
                version
                status
            }
            problems {
                message
                field
                type
            }
        }
    }
`;

// UploadPackageVersion PUTs the tar.gz bytes to the package version upload endpoint.
// Shared by the create-version and edit-version flows (a re-upload to an existing version id
// overwrites its package in place).
export async function UploadPackageVersion(
    authService: AuthenticationService,
    versionId: string,
    bytes: Uint8Array<ArrayBuffer>
): Promise<void> {
    const response = await authService.fetchWithAuth( // nosemgrep: nodejs_scan.javascript-ssrf-rule-node_ssrf
        `${cfg.apiUrl}/v1/package-registry/versions/${versionId}/upload`,
        {
            method: 'PUT',
            headers: { 'Content-Type': 'application/octet-stream' },
            body: bytes,
        }
    );

    if (!response.ok) {
        throw new Error(`request to upload package version returned status ${response.status}`);
    }
}

function NewPackageVersion() {
    const { packageId } = useParams();

    const queryData = useLazyLoadQuery<NewPackageVersionQuery>(
        query,
        { id: packageId as string },
        { fetchPolicy: 'store-and-network' }
    );

    return (
        <Box flexGrow={1} minWidth={0}>
            {queryData.node?.name !== undefined && <NewPackageVersionForm
                packageId={queryData.node.id!}
                name={queryData.node.name}
                groupPath={queryData.node.groupPath!}
                kind={queryData.node.kind!}
                seedVersionId={queryData.node.latestVersion?.status === 'UPLOADED'
                    ? queryData.node.latestVersion.id
                    : undefined}
            />}
            {queryData.node?.name === undefined && <Box display="flex" justifyContent="center" marginTop={4}>
                <Typography variant="h6" color="textSecondary">
                    package not found
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
    // seedVersionId is the version whose files the editor opens with, when there is a published one to
    // copy. Absent for a package's first version, or while its only version is still uploading.
    seedVersionId?: string;
}

function NewPackageVersionForm({ packageId, name, groupPath, kind, seedVersionId }: FormProps) {
    // The group shell owns the page layout, so widen it rather than nesting another provider
    // (a nested provider would still be capped by the outer max width).
    usePageLayout('fullscreen');
    const navigate = useNavigate();
    const authService = useContext<AuthenticationService>(AuthServiceContext);
    const { headerHeight } = useAppHeaderHeight();

    const [version, setVersion] = useState('');
    const [showSampleInput, setShowSampleInput] = useState(false);

    // files is null until the seed has been resolved: either the latest version's files have been
    // downloaded and extracted, or there is nothing to copy and the starter policy is used.
    const [files, setFiles] = useState<PolicyFile[] | null>(null);
    const [error, setError] = useState<MutationError>();
    const [uploading, setUploading] = useState(false);

    useEffect(() => {
        if (!seedVersionId) {
            setFiles([newPolicyFile('policy.rego', SAMPLE_OPA_POLICY)]);
            return;
        }

        let cancelled = false;
        fetchPolicyFiles(authService, seedVersionId)
            .then(loaded => {
                if (cancelled) {
                    return;
                }
                // An empty package is an edge case; fall back to the starter policy so the editor is usable.
                setFiles(loaded.length > 0 ? loaded : [newPolicyFile('policy.rego', SAMPLE_OPA_POLICY)]);
            })
            .catch(downloadError => {
                if (cancelled) {
                    return;
                }
                // The seed is a convenience, so a failed download starts from the starter policy rather
                // than blocking the page.
                setError({
                    severity: 'warning',
                    message: `failed to load the latest version's files: ${downloadError.message}`
                });
                setFiles([newPolicyFile('policy.rego', SAMPLE_OPA_POLICY)]);
            });

        return () => { cancelled = true; };
    }, [authService, seedVersionId]);

    const [commitCreate, createInFlight] = useMutation<NewPackageVersionCreateMutation>(createMutation);

    const canPublish = version.trim() !== '' && files != null &&
        files.some(file => file.name.trim().toLowerCase().endsWith('.rego') && file.content.trim() !== '');

    const onPublish = async () => {
        if (files == null) {
            return;
        }
        setError(undefined);
        setUploading(true);
        try {
            // Build the tar.gz bytes from the authored files. Copy into a fresh
            // ArrayBuffer-backed Uint8Array so it satisfies BodyInit/BufferSource.
            const bytes = new Uint8Array(await createTarGzip(files.map(file => ({
                name: file.name,
                data: new window.TextEncoder().encode(file.content),
            }))));

            // Hash exactly the uploaded bytes (SHA-256, hex-encoded).
            const digest = await window.crypto.subtle.digest('SHA-256', bytes);
            const shaSum = [...new Uint8Array(digest)].map(b => b.toString(16).padStart(2, '0')).join('');

            commitCreate({
                variables: {
                    input: {
                        packageId,
                        version: version.trim(),
                        shaSum,
                    }
                },
                onCompleted: data => {
                    if (data.createPackageVersion.problems.length) {
                        setUploading(false);
                        setError({
                            severity: 'warning',
                            message: data.createPackageVersion.problems.map(problem => problem.message).join('; ')
                        });
                    } else if (!data.createPackageVersion.packageVersion) {
                        setUploading(false);
                        setError({
                            severity: 'error',
                            message: 'Unexpected error occurred'
                        });
                    } else {
                        const versionId = data.createPackageVersion.packageVersion.id;
                        UploadPackageVersion(authService, versionId, bytes)
                            .then(() => {
                                navigate(`/groups/${groupPath}/-/packages/${packageId}?tab=versions`);
                            })
                            .catch(uploadError => {
                                setUploading(false);
                                setError({
                                    severity: 'warning',
                                    message: `failed to upload package version: ${uploadError.message}`
                                });
                            });
                    }
                },
                onError: mutationError => {
                    setUploading(false);
                    setError({
                        severity: 'error',
                        message: `Unexpected error occurred: ${mutationError.message}`
                    });
                }
            });
        } catch (buildError: any) {
            setUploading(false);
            setError({
                severity: 'error',
                message: `failed to build package package: ${buildError.message}`
            });
        }
    };

    const busy = createInFlight || uploading;

    return (
        <Box sx={{ display: 'flex', flexDirection: 'column', height: `calc(100vh - ${headerHeight}px - 32px)` }}>
            <Box marginBottom={2} display="flex" justifyContent="space-between" alignItems="flex-start" gap={2}>
                <Box minWidth={0}>
                    <Box display="flex" alignItems="center" gap={1}>
                        <Typography variant="h6">{name}</Typography>
                        <Chip variant="outlined" size="small" label="new version" />
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
                placeholder="e.g. 1.0.0"
                onChange={event => setVersion(event.target.value)}
            />

            <Box sx={{ flexGrow: 1, minHeight: 0, marginTop: 2, display: 'flex', gap: 2 }}>
                <Box sx={{ flex: 1, minWidth: 0 }}>
                    {files == null
                        ? <Box display="flex" justifyContent="center" padding={4}><CircularProgress /></Box>
                        : <PackageFilesEditor files={files} onChange={setFiles} />}
                </Box>

                {kind === 'OPA_POLICY' && showSampleInput && <OPASampleInputsPanel />}
            </Box>

            <Box marginTop={2} display="flex" alignItems="center" justifyContent="space-between" gap={2}>
                <Box>
                    <Button
                        sx={{ marginRight: 2 }}
                        loading={busy}
                        disabled={!canPublish}
                        variant="outlined"
                        color="primary"
                        onClick={onPublish}
                    >
                        Publish
                    </Button>
                    <Button component={RouterLink} color="inherit" to={`/groups/${groupPath}/-/packages/${packageId}?tab=versions`}>Cancel</Button>
                </Box>
                {kind === 'OPA_POLICY' && (
                    // An authoring aid, not a form action, so it sits opposite Publish rather than beside
                    // it — reachable the whole time the policy is being written without reading as a third
                    // button. A new tab, so the unsaved editor survives the detour.
                    <Link
                        href="https://play.openpolicyagent.org/"
                        target="_blank"
                        rel="noopener noreferrer"
                        underline="hover"
                        variant="body2"
                        sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5, whiteSpace: 'nowrap' }}
                    >
                        Open OPA Playground
                        <OpenInNewIcon sx={{ width: 14, height: 14 }} />
                    </Link>
                )}
            </Box>
        </Box>
    );
}

export default NewPackageVersion;
