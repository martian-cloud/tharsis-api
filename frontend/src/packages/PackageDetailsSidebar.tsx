import { Box, Paper, Tooltip, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { ReactNode } from 'react';
import { useFragment } from 'react-relay/hooks';
import CopyButton from '../common/CopyButton';
import Gravatar from '../common/Gravatar';
import Timestamp from '../common/Timestamp';
import Link from '../routes/Link';
import { PackageDetailsSidebarFragment_package$key } from './__generated__/PackageDetailsSidebarFragment_package.graphql';
import { PackageDetailsSidebarFragment_version$key } from './__generated__/PackageDetailsSidebarFragment_version.graphql';

export const SidebarWidth = 340;

// The API stores a kind as an enum; spell it the way the registry talks about it.
const PACKAGE_KIND_LABELS: Record<string, string> = {
    OPA_POLICY: 'Rego policy',
};

const VISIBILITY_LABELS: Record<string, string> = {
    GLOBAL: 'Global',
    PRIVATE: 'Private',
    ROOT_GROUP: 'Root group',
};

// formatBytes renders a package size. A size of 0 means unknown rather than empty — it is the default
// for versions uploaded before the field existed, and for versions whose upload hasn't finished — so
// it reads as an absent value instead of "0 B".
function formatBytes(size: number): string {
    if (size <= 0) {
        return '—';
    }
    if (size < 1024) {
        return `${size} B`;
    }
    const units = ['KB', 'MB', 'GB'];
    let value = size / 1024;
    let unit = 0;
    while (value >= 1024 && unit < units.length - 1) {
        value /= 1024;
        unit++;
    }
    return `${value.toFixed(1)} ${units[unit]}`;
}

const SECTION_LABEL_SX = {
    fontWeight: 700,
    textTransform: 'uppercase',
    letterSpacing: '0.09em',
} as const;

// Row lays a field out label-left / value-right. The value column can shrink and wrap so a long
// value never pushes its label out of the card.
function Row({ label, value, mono }: { label: string, value: ReactNode, mono?: boolean }) {
    const theme = useTheme();
    return (
        <Box sx={{ display: 'flex', justifyContent: 'space-between', gap: 2, padding: '8px 16px' }}>
            <Typography variant="body2" component="div" sx={{ color: theme.palette.text.secondary, flexShrink: 0 }}>
                {label}
            </Typography>
            <Typography
                variant={mono ? 'code' : 'body2'}
                component="div"
                sx={{
                    color: theme.palette.text.primary,
                    textAlign: 'right',
                    minWidth: 0,
                    wordBreak: 'break-word',
                }}
            >
                {value}
            </Typography>
        </Box>
    );
}

interface Props {
    fragmentRef: PackageDetailsSidebarFragment_package$key;
    // versionFragmentRef is the version being viewed, which is the latest version unless an earlier one
    // was picked from the version history. Null when the package has no versions at all.
    versionFragmentRef: PackageDetailsSidebarFragment_version$key | null;
}

// PackageDetailsSidebar describes the package and the version being viewed alongside the page's tab
// content — the same version whose files the Files tab browses.
function PackageDetailsSidebar(props: Props) {
    const theme = useTheme();

    const data = useFragment<PackageDetailsSidebarFragment_package$key>(
        graphql`
          fragment PackageDetailsSidebarFragment_package on Package
          {
              visibility
              kind
              groupPath
              allowMutableVersions
          }
        `,
        props.fragmentRef
    );

    const latest = useFragment<PackageDetailsSidebarFragment_version$key>(
        graphql`
          fragment PackageDetailsSidebarFragment_version on PackageVersion
          {
              version
              latest
              createdBy
              shaSum
              size
              metadata {
                  createdAt
              }
          }
        `,
        props.versionFragmentRef
    );

    return (
        <Paper variant="outlined" sx={{ borderRadius: '12px', overflow: 'hidden' }}>
            <Typography
                variant="caption"
                component="div"
                sx={{ ...SECTION_LABEL_SX, color: theme.palette.text.secondary, padding: '16px 16px 8px 16px' }}
            >
                Details
            </Typography>
            {latest && <>
                <Row
                    label="Published"
                    value={<Timestamp component="span" timestamp={latest.metadata.createdAt} />}
                />
                <Row
                    label="Published by"
                    value={
                        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'flex-end', gap: 1 }}>
                            <Tooltip title={latest.createdBy}>
                                <Box sx={{ display: 'flex' }}>
                                    <Gravatar width={20} height={20} email={latest.createdBy} />
                                </Box>
                            </Tooltip>
                            <Box component="span" sx={{ minWidth: 0, wordBreak: 'break-word' }}>{latest.createdBy}</Box>
                        </Box>
                    }
                />
                <Row label="Version" value={latest.version} mono />
            </>}
            <Row label="Visibility" value={VISIBILITY_LABELS[data.visibility] ?? data.visibility} />
            <Row label="Type" value={PACKAGE_KIND_LABELS[data.kind] ?? data.kind} />
            {latest && <Row label="Size" value={formatBytes(latest.size)} />}
            <Row label="Allow mutable versions" value={data.allowMutableVersions ? 'Yes' : 'No'} />
            {/* The group isn't in the design, but the header no longer carries the package's path and
                the registry page's breadcrumbs don't either, so it would otherwise be lost. */}
            <Row
                label="Group"
                value={<Link underline="hover" to={`/groups/${data.groupPath}`}>{data.groupPath}</Link>}
            />
            <Box sx={{ height: '1px', background: theme.palette.divider, mt: 1 }} />
            {latest ? (
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1, padding: '16px' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 1 }}>
                        <Typography variant="body2" component="div" sx={{ color: theme.palette.text.secondary }}>
                            Digest
                        </Typography>
                        <CopyButton data={latest.shaSum} toolTip="Copy digest" />
                    </Box>
                    <Typography
                        variant="code"
                        component="div"
                        sx={{
                            lineHeight: 1.6,
                            color: theme.palette.text.primary,
                            wordBreak: 'break-all',
                        }}
                    >
                        {latest.shaSum}
                    </Typography>
                </Box>
            ) : (
                <Box sx={{ padding: '16px' }}>
                    <Typography variant="body2" sx={{ color: theme.palette.text.secondary }}>
                        No versions have been published yet.
                    </Typography>
                </Box>
            )}
        </Paper>
    );
}

export default PackageDetailsSidebar;
