import { Alert, Box, CircularProgress, Link as MuiLink, Tab, Tabs, Typography, useTheme } from '@mui/material';
import { alpha } from '@mui/material/styles';
import graphql from 'babel-plugin-relay/macro';
import React, { ReactNode, Suspense, useCallback } from 'react';
import { useFragment, useLazyLoadQuery, useRefetchableFragment } from 'react-relay/hooks';
import { useSearchParams } from 'react-router-dom';
import ListSkeleton from '../skeletons/ListSkeleton';
import PackageDetailsSidebar, { SidebarWidth } from './PackageDetailsSidebar';
import PackageVersionActions from './PackageVersionActions';
import PackageVersionFiles from './PackageVersionFiles';
import PackageVersionList from './PackageVersionList';
import { PackageDetailsIndexFragment_package$data, PackageDetailsIndexFragment_package$key } from './__generated__/PackageDetailsIndexFragment_package.graphql';
import { PackageDetailsIndexFragment_version$key } from './__generated__/PackageDetailsIndexFragment_version.graphql';
import { PackageDetailsIndexRefetchQuery } from './__generated__/PackageDetailsIndexRefetchQuery.graphql';
import { PackageDetailsIndexSelectedVersionQuery } from './__generated__/PackageDetailsIndexSelectedVersionQuery.graphql';

// The version being viewed is either the package's latest version or one picked out of the version
// history, so both paths select the same fields through this fragment.
const versionFragment = graphql`
  fragment PackageDetailsIndexFragment_version on PackageVersion
  {
      id
      version
      status
      error
      latest
      ...PackageDetailsSidebarFragment_version
  }
`;

// An earlier version is addressed by id rather than reached through the package's versions connection:
// the connection is paginated, so a linked-to version isn't necessarily loaded.
const selectedVersionQuery = graphql`
    query PackageDetailsIndexSelectedVersionQuery($id: String!) {
        node(id: $id) {
            ... on PackageVersion {
                ...PackageDetailsIndexFragment_version
            }
        }
    }
`;

interface Props {
    fragmentRef: PackageDetailsIndexFragment_package$key;
    // The two pages reach a package by different routes, and only the group page can create versions,
    // so the breadcrumb trail and the header actions are supplied by the caller.
    breadcrumbs: ReactNode;
    actions: ReactNode;
    // canManageVersions adds the create / edit / delete control to the header. Those act on the version
    // being viewed, which only this component knows, so it renders the control itself rather than taking
    // it through `actions`. The registry page is read-only and leaves it off.
    canManageVersions?: boolean;
}

// PackageDetailsIndex is the shared body of the group and registry package pages. Both show the same
// package, one version's files, and its version history — they differ only in how they were navigated
// to and in whether versions can be created, edited or deleted.
function PackageDetailsIndex(props: Props) {
    const [searchParams, setSearchParams] = useSearchParams();

    const [data, refetch] = useRefetchableFragment<PackageDetailsIndexRefetchQuery, PackageDetailsIndexFragment_package$key>(
        graphql`
          fragment PackageDetailsIndexFragment_package on Package
          @refetchable(queryName: "PackageDetailsIndexRefetchQuery")
          {
              id
              name
              description
              groupPath
              allowMutableVersions
              latestVersion {
                  id
                  ...PackageDetailsIndexFragment_version
              }
              ...PackageDetailsSidebarFragment_package
              ...PackageVersionListFragment_package
          }
        `,
        props.fragmentRef
    );

    const latest = data.latestVersion;
    // The latest version is canonical, so it is viewed without a version param; anything else is an
    // explicit selection out of the version history.
    const versionParam = searchParams.get('version');
    const selectedVersionId = versionParam && versionParam !== latest?.id ? versionParam : null;

    // updateParams merges into the existing query string: the tab, the selected version and the file
    // the archive browser tracks all live there, and none of them may drop the others.
    const updateParams = useCallback((mutate: (params: URLSearchParams) => void) => {
        setSearchParams(prev => {
            const next = new URLSearchParams(prev);
            mutate(next);
            return next;
        }, { replace: true });
    }, [setSearchParams]);

    const onSelectVersion = useCallback((versionId: string) => {
        updateParams(params => {
            if (versionId === latest?.id) {
                params.delete('version');
            } else {
                params.set('version', versionId);
            }
            // Selecting a version is a request to look at it, so move to the files it contains.
            params.set('tab', 'files');
            // The selected file is per-version; a path from the previous version may not exist here.
            params.delete('file');
            params.delete('line');
        });
    }, [updateParams, latest?.id]);

    const onViewLatest = useCallback(() => {
        updateParams(params => {
            params.delete('version');
            params.delete('file');
            params.delete('line');
        });
    }, [updateParams]);

    // Deleting a version may have promoted a new latest, which the mutation response can't carry, so
    // re-read the package. Anything pointing at the deleted version has to let go of it first.
    const onVersionDeleted = useCallback((versionId: string) => {
        if (versionId === versionParam) {
            onViewLatest();
        }
        refetch({}, { fetchPolicy: 'network-only' });
    }, [versionParam, onViewLatest, refetch]);

    const body = {
        pkg: data,
        breadcrumbs: props.breadcrumbs,
        actions: props.actions,
        canManageVersions: props.canManageVersions,
        onSelectVersion,
        onViewLatest,
        onVersionDeleted,
        tab: searchParams.get('tab'),
        onTabChange: (tab: string) => updateParams(params => params.set('tab', tab)),
    };

    // The boundary is mounted for both branches so switching versions keeps the current page on screen
    // while the selected version loads, rather than falling back to the spinner.
    return (
        <Suspense fallback={
            <Box padding={4} display="flex" justifyContent="center">
                <CircularProgress />
            </Box>
        }>
            {selectedVersionId
                ? <SelectedVersionBody versionId={selectedVersionId} {...body} />
                : <PackageDetailsBody versionFragmentRef={latest ?? null} {...body} />}
        </Suspense>
    );
}

interface BodyProps {
    pkg: PackageDetailsIndexFragment_package$data;
    breadcrumbs: ReactNode;
    actions: ReactNode;
    canManageVersions?: boolean;
    onSelectVersion: (versionId: string) => void;
    onViewLatest: () => void;
    onVersionDeleted: (versionId: string) => void;
    tab: string | null;
    onTabChange: (tab: string) => void;
}

function SelectedVersionBody({ versionId, ...props }: BodyProps & { versionId: string }) {
    const queryData = useLazyLoadQuery<PackageDetailsIndexSelectedVersionQuery>(
        selectedVersionQuery,
        { id: versionId },
        { fetchPolicy: 'store-and-network' },
    );

    return <PackageDetailsBody versionFragmentRef={queryData.node ?? null} {...props} />;
}

function PackageDetailsBody(props: BodyProps & { versionFragmentRef: PackageDetailsIndexFragment_version$key | null }) {
    const { pkg, breadcrumbs, actions, canManageVersions, onSelectVersion, onViewLatest, onVersionDeleted, onTabChange } = props;
    const theme = useTheme();

    const version = useFragment<PackageDetailsIndexFragment_version$key>(versionFragment, props.versionFragmentRef);

    // A package always has a latest version once anything has been published, so this also decides
    // whether there are any files to browse at all.
    const hasVersions = pkg.latestVersion != null;
    // With nothing published there is no Files tab to select, so a stale ?tab=files can't be honored.
    const tab = hasVersions ? (props.tab || 'files') : 'versions';

    const filesAvailable = version?.status === 'UPLOADED';
    const uploading = version?.status === 'PENDING' || version?.status === 'UPLOAD_IN_PROGRESS';

    return (
        <Box>
            {breadcrumbs}
            <Box
                sx={{
                    display: 'flex',
                    marginBottom: 3,
                    justifyContent: 'space-between',
                    flexDirection: { xs: 'column', md: 'row' },
                    alignItems: { xs: 'flex-start', md: 'flex-start' },
                    gap: 3,
                }}
            >
                <Box sx={{ minWidth: 0 }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, flexWrap: 'wrap' }}>
                        <Typography
                            variant="h5"
                            component="h1"
                            sx={{ m: 0, fontWeight: 600, color: theme.palette.text.primary }}
                        >
                            {pkg.name}
                        </Typography>
                        {version && <Box
                            component="span"
                            sx={{
                                display: 'inline-flex',
                                padding: '3px 10px',
                                borderRadius: '999px',
                                fontFamily: theme.typography.code.fontFamily,
                                fontSize: theme.typography.caption.fontSize,
                                fontWeight: 600,
                                color: theme.palette.primary.main,
                                background: alpha(theme.palette.primary.main, 0.12),
                                border: `1px solid ${alpha(theme.palette.primary.main, 0.35)}`,
                            }}
                        >
                            v{version.version}
                        </Box>}
                        {version?.latest && <Box
                            component="span"
                            sx={{
                                display: 'inline-flex',
                                padding: '3px 10px',
                                borderRadius: '999px',
                                fontSize: theme.typography.caption.fontSize,
                                fontWeight: 600,
                                letterSpacing: '0.06em',
                                color: theme.palette.success.main,
                                background: alpha(theme.palette.success.main, 0.12),
                            }}
                        >
                            LATEST
                        </Box>}
                    </Box>
                    {pkg.description && <Typography sx={{ mt: 1, color: theme.palette.text.secondary }}>
                        {pkg.description}
                    </Typography>}
                </Box>
                <Box sx={{ display: 'flex', gap: 1.5, flexShrink: 0, alignItems: 'center' }}>
                    {canManageVersions && <PackageVersionActions
                        packageId={pkg.id}
                        groupPath={pkg.groupPath}
                        allowMutableVersions={pkg.allowMutableVersions}
                        version={version ? { id: version.id, version: version.version, status: version.status } : null}
                        onVersionDeleted={onVersionDeleted}
                    />}
                    {actions}
                </Box>
            </Box>
            {version && !version.latest && <Alert sx={{ marginBottom: 2 }} severity="info">
                You are viewing an earlier version of this package.
                {' '}
                <MuiLink component="button" underline="hover" sx={{ fontFamily: 'inherit', fontSize: 'inherit' }} onClick={onViewLatest}>
                    view latest version
                </MuiLink>
            </Alert>}
            {uploading && <Alert sx={{ marginBottom: 2 }} severity="warning">
                Upload is still in progress
            </Alert>}
            {version?.status === 'ERRORED' && <Alert sx={{ marginBottom: 2 }} severity="error">
                {version.error || 'Upload failed'}
            </Alert>}
            <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
                <Tabs
                    value={tab}
                    onChange={(_event: React.SyntheticEvent, newValue: string) => onTabChange(newValue)}
                    variant="scrollable"
                    scrollButtons="auto"
                    allowScrollButtonsMobile
                >
                    {/* A package with no versions has no files, so the tab is left out entirely rather
                        than opening onto an empty viewer. */}
                    {hasVersions && <Tab label="Files" value="files" />}
                    <Tab label="Versions" value="versions" />
                </Tabs>
            </Box>
            {/* The sidebar sits beside both tabs, not just Files, so the content column doesn't
                change width when switching tabs. Below lg it stacks under the content rather than
                squeezing the file viewer to nothing. */}
            <Box
                sx={{
                    display: 'grid',
                    gridTemplateColumns: `minmax(0, 1fr) ${SidebarWidth}px`,
                    // Grid items stretch by default, which would run the details card down the whole
                    // height of the file viewer. It should be only as tall as its own content.
                    alignItems: 'start',
                    gap: 4,
                    paddingTop: 3,
                    [theme.breakpoints.down('lg')]: { gridTemplateColumns: 'minmax(0, 1fr)' },
                }}
            >
                <Box sx={{ minWidth: 0 }}>
                    {tab === 'files' && (filesAvailable && version
                        ? <PackageVersionFiles versionId={version.id} />
                        : <Box padding={2} display="flex" justifyContent="center" alignItems="center">
                            <Typography color="textSecondary">No files are available for this version.</Typography>
                        </Box>)}
                    {tab === 'versions' && <Suspense fallback={<ListSkeleton rowCount={3} />}>
                        <PackageVersionList
                            fragmentRef={pkg}
                            onSelectVersion={onSelectVersion}
                        />
                    </Suspense>}
                </Box>
                {/* The sidebar's version fragment is spread inside this component's version fragment,
                    so it is reached through the read data rather than the raw ref. */}
                <PackageDetailsSidebar fragmentRef={pkg} versionFragmentRef={version ?? null} />
            </Box>
        </Box>
    );
}

export default PackageDetailsIndex;
