import CopyIcon from '@mui/icons-material/ContentCopy';
import { Stack, Tooltip, Typography } from '@mui/material';
import IconButton from '@mui/material/IconButton';
import graphql from 'babel-plugin-relay/macro';
import { useContext, useMemo } from 'react';
import { useFragment } from 'react-relay/hooks';
import { ApiConfigContext } from '../ApiConfigContext';
import Link from '../routes/Link';
import { ModuleSourceLinkFragment_run$key } from './__generated__/ModuleSourceLinkFragment_run.graphql';

interface Props {
    fragmentRef: ModuleSourceLinkFragment_run$key
    // Non-Tharsis module sources are shown as plain text (they aren't navigable within Tharsis) and
    // are truncated to this many characters, with the full value in a tooltip. Tharsis-registry
    // modules are always shown in full since they're a link, not free text.
    truncateAt?: number
}

// Renders a run's module source as a link into the module registry when it references a module in
// this Tharsis instance's own registry, or as plain (optionally truncated) text with a copy button
// otherwise -- external module sources aren't navigable within Tharsis. Shared by RunDetailsSidebar
// and WorkspaceDetailsIndex so the tharsis-vs-external detection and link construction live in one
// place.
function ModuleSourceLink({ fragmentRef, truncateAt }: Props) {
    const apiConfig = useContext(ApiConfigContext);

    const data = useFragment<ModuleSourceLinkFragment_run$key>(graphql`
        fragment ModuleSourceLinkFragment_run on Run {
            moduleSource
            moduleVersion
        }
    `, fragmentRef);

    // If module source references a module in the tharsis registry than strip the host
    const moduleSource = useMemo(
        () => (data.moduleSource && data.moduleSource.startsWith(apiConfig.serviceDiscoveryHost)) ? data.moduleSource.substring(apiConfig.serviceDiscoveryHost.length + 1) : data.moduleSource,
        [data.moduleSource, apiConfig.serviceDiscoveryHost]
    );

    const isTharsisModule = useMemo(() => !!moduleSource && moduleSource.length !== data.moduleSource?.length, [moduleSource, data.moduleSource]);

    if (!moduleSource) {
        return null;
    }

    if (isTharsisModule) {
        return (
            <Tooltip title={moduleSource}>
                <Typography color="secondary" component="span" noWrap>
                    <Link color="inherit" noWrap underline="hover" to={`/module-registry/${moduleSource}/${data.moduleVersion}`}>
                        {moduleSource}
                    </Link>
                </Typography>
            </Tooltip>
        );
    }

    const displayText = truncateAt && moduleSource.length > truncateAt
        ? `${moduleSource.substring(0, truncateAt)}...`
        : moduleSource;

    return (
        <Stack direction="row" spacing={1} alignItems="center">
            <Tooltip title={data.moduleSource}>
                <Typography sx={{ wordBreak: 'break-all' }}>
                    {displayText}
                </Typography>
            </Tooltip>
            <IconButton sx={{ padding: '4px' }} onClick={() => navigator.clipboard.writeText(data.moduleSource ?? '')}>
                <CopyIcon sx={{ width: 16, height: 16 }} />
            </IconButton>
        </Stack>
    );
}

export default ModuleSourceLink;
