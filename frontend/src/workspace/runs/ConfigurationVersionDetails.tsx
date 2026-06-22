import DownloadIcon from '@mui/icons-material/Download';
import { Alert, Box, Button, Stack, Tooltip, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import { useCallback, useContext, useRef } from 'react';
import { useFragment, useLazyLoadQuery } from 'react-relay/hooks';
import { useParams } from 'react-router-dom';
import AuthServiceContext from '../../auth/AuthServiceContext';
import AuthenticationService from '../../auth/AuthenticationService';
import ArchiveFileBrowser from '../../archive/ArchiveFileBrowser';
import { ArchiveTooLargeError, MAX_DOWNLOAD_BYTES } from '../../archive/tarball';
import cfg from '../../common/config';
import downloadFile from '../../common/filedownload';
import Gravatar from '../../common/Gravatar';
import Timestamp from '../../common/Timestamp';
import TRNButton from '../../common/TRNButton';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import { ConfigurationVersionDetailsFragment_workspace$key } from './__generated__/ConfigurationVersionDetailsFragment_workspace.graphql';
import { ConfigurationVersionDetailsQuery } from './__generated__/ConfigurationVersionDetailsQuery.graphql';

// fetchConfigurationVersionPackage downloads the configuration version tarball as an ArrayBuffer.
async function fetchConfigurationVersionPackage(authService: AuthenticationService, id: string): Promise<ArrayBuffer> {
    const response = await authService.fetchWithAuth(`${cfg.apiUrl}/tfe/v2/configuration-versions/${id}/content`, {
        method: 'GET',
    });

    if (!response.ok) {
        throw new Error(`request for configuration version content returned status ${response.status}`);
    }

    const contentLength = Number(response.headers.get('content-length'));
    if (!contentLength) {
        throw new Error('configuration version archive response is missing a content-length header');
    }

    if (contentLength > MAX_DOWNLOAD_BYTES) {
        throw new ArchiveTooLargeError('configuration version archive is too large to preview');
    }

    return response.arrayBuffer();
}

interface Props {
    fragmentRef: ConfigurationVersionDetailsFragment_workspace$key;
}

function ConfigurationVersionDetails({ fragmentRef }: Props) {
    const { id } = useParams();
    const configurationVersionId = id as string;

    const authService = useContext<AuthenticationService>(AuthServiceContext);
    const { enqueueSnackbar } = useSnackbar();

    const workspace = useFragment<ConfigurationVersionDetailsFragment_workspace$key>(graphql`
        fragment ConfigurationVersionDetailsFragment_workspace on Workspace {
            fullPath
        }
    `, fragmentRef);

    const queryData = useLazyLoadQuery<ConfigurationVersionDetailsQuery>(graphql`
        query ConfigurationVersionDetailsQuery($id: String!) {
            node(id: $id) {
                ... on ConfigurationVersion {
                    id
                    status
                    createdBy
                    metadata {
                        createdAt
                        trn
                    }
                }
            }
        }
    `, { id: configurationVersionId }, { fetchPolicy: 'store-and-network' });

    const cachedBuffer = useRef<ArrayBuffer | null>(null);

    const load = useCallback(async () => {
        const buffer = await fetchConfigurationVersionPackage(authService, configurationVersionId);
        cachedBuffer.current = buffer;
        return buffer;
    }, [authService, configurationVersionId]);

    const onDownload = async () => {
        try {
            const buffer = cachedBuffer.current ?? await load();
            downloadFile(`${configurationVersionId}.tar.gz`, new Blob([buffer]));
        } catch (error) {
            console.error('failed to download configuration version', error);
            enqueueSnackbar('failed to download configuration version', { variant: 'error' });
        }
    };

    if (!queryData.node) {
        return <Box>Not Found</Box>;
    }

    const createdBy = queryData.node.createdBy ?? '';
    const status = queryData.node.status ?? '';
    const filesAvailable = status === 'uploaded';

    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={workspace.fullPath}
                childRoutes={[
                    { title: 'configuration versions', path: 'configuration_versions', disabled: true },
                    { title: `${configurationVersionId.substring(0, 8)}...`, path: configurationVersionId }
                ]}
            />
            {status === 'pending' && <Alert sx={{ marginBottom: 2 }} severity="warning">
                Upload is still in progress
            </Alert>}
            {status === 'errored' && <Alert sx={{ marginBottom: 2 }} severity="error">
                Upload failed for this configuration version
            </Alert>}
            <Box display="flex" justifyContent="space-between" alignItems="center" marginBottom={2}>
                <Stack direction="row" spacing={0.75} alignItems="center">
                    <Typography component="div">Configuration version created</Typography>
                    <Timestamp timestamp={queryData.node.metadata?.createdAt as string} />
                    <Typography component="div">by</Typography>
                    <Tooltip title={createdBy}>
                        <Box>
                            <Gravatar width={20} height={20} email={createdBy} />
                        </Box>
                    </Tooltip>
                </Stack>
                <Stack direction="row" spacing={1} alignItems="center">
                    <TRNButton trn={queryData.node.metadata?.trn ?? ''} size="small" />
                    {filesAvailable && <Button size="small" color="info" variant="outlined" startIcon={<DownloadIcon />} onClick={onDownload}>
                        Download
                    </Button>}
                </Stack>
            </Box>
            {filesAvailable
                ? <ArchiveFileBrowser load={load} preferredFile="main.tf" />
                : <Box padding={2} display="flex" justifyContent="center" alignItems="center">
                    <Typography color="textSecondary">No files are available for this version.</Typography>
                </Box>}
        </Box>
    );
}

export default ConfigurationVersionDetails;
