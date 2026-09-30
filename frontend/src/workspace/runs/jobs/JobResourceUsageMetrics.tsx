import DeveloperBoardIcon from '@mui/icons-material/DeveloperBoard';
import MemoryIcon from '@mui/icons-material/Memory';
import NorthIcon from '@mui/icons-material/North';
import SouthIcon from '@mui/icons-material/South';
import StorageIcon from '@mui/icons-material/Storage';
import { Box, Typography } from '@mui/material';
import { useTheme } from '@mui/material/styles';
import graphql from 'babel-plugin-relay/macro';
import { ReactNode } from 'react';
import { useFragment } from 'react-relay/hooks';
import { shortDuration } from '../../../common/duration';
import RunDetailsStageTabEmptyState from '../RunDetailsStageTabEmptyState';
import JobMetricTile from './JobMetricTile';
import { JobResourceUsageMetricsFragment_job$key } from './__generated__/JobResourceUsageMetricsFragment_job.graphql';
import { formatBytes } from './jobUtils';

// ByteMetricTile is a byte-valued JobMetricTile: it formats usage/limit as byte sizes and uses the
// default 80/90/95 color bands for the usage bar.
function ByteMetricTile({ icon, label, value, limit }: {
    icon: ReactNode;
    label: string;
    value: number;
    limit?: number | null;
}) {
    return (
        <JobMetricTile
            icon={icon}
            label={label}
            value={formatBytes(value)}
            usage={value}
            limit={limit}
            format={formatBytes}
        />
    );
}

// MetricGroup is a labeled cluster of related tiles (e.g. Compute, Network, Disk). It renders
// nothing when none of its tiles have data, so an all-empty group leaves no stray heading.
function MetricGroup({ label, children }: { label: string; children: ReactNode }) {
    const theme = useTheme();

    if (!Array.isArray(children) ? !children : !children.some(Boolean)) {
        return null;
    }

    return (
        <Box sx={{ display: 'flex', flexDirection: 'column' }}>
            <Typography
                variant="caption"
                sx={{ display: 'block', mb: 1, fontWeight: 600, letterSpacing: '0.06em', textTransform: 'uppercase', color: theme.palette.text.secondary }}
            >
                {label}
            </Typography>
            <Box sx={{ flex: 1, display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(130px, 1fr))', gap: 2 }}>
                {children}
            </Box>
        </Box>
    );
}

// JobResourceUsageMetrics renders a job's resource usage as a grid of metric groups, without card
// chrome so it can sit directly in a run stage tab (JobDetailsResourceUsageCard wraps it elsewhere).
function JobResourceUsageMetrics({ fragmentRef, emptyMessage, columns = 'auto' }: {
    fragmentRef: JobResourceUsageMetricsFragment_job$key;
    // When set, an absent metrics object renders this message instead of nothing.
    emptyMessage?: string;
    // 'auto' fits groups side by side (job details page); 'single' stacks them (narrow tab panel).
    columns?: 'auto' | 'single';
}) {
    const job = useFragment(
        graphql`
        fragment JobResourceUsageMetricsFragment_job on Job {
            resourceUsageMetrics {
                peakMemoryBytes
                totalCpuTimeMs
                totalNetworkReceivedBytes
                totalNetworkSentBytes
                totalDiskReadBytes
                totalDiskWriteBytes
            }
            resourceUsageLimits {
                memoryBytes
                networkReceivedBytes
                networkSentBytes
                diskReadBytes
                diskWriteBytes
            }
        }
        `,
        fragmentRef
    );

    const metrics = job.resourceUsageMetrics;
    const limits = job.resourceUsageLimits;

    if (!metrics) {
        return emptyMessage ? <RunDetailsStageTabEmptyState message={emptyMessage} /> : null;
    }

    return (
        <Box sx={{
            display: 'grid',
            gridTemplateColumns: columns === 'single' ? '1fr' : 'repeat(auto-fit, minmax(280px, 1fr))',
            gap: 3,
        }}>
            <MetricGroup label="Compute">
                {metrics.totalCpuTimeMs != null && (
                    <JobMetricTile icon={<DeveloperBoardIcon fontSize="small" />} label="CPU time" value={shortDuration(metrics.totalCpuTimeMs)} />
                )}
                {metrics.peakMemoryBytes != null && (
                    <ByteMetricTile
                        icon={<MemoryIcon fontSize="small" />}
                        label="Peak memory"
                        value={metrics.peakMemoryBytes}
                        limit={limits?.memoryBytes}
                    />
                )}
            </MetricGroup>

            <MetricGroup label="Network">
                {metrics.totalNetworkReceivedBytes != null && (
                    <ByteMetricTile
                        icon={<SouthIcon fontSize="small" />}
                        label="Received"
                        value={metrics.totalNetworkReceivedBytes}
                        limit={limits?.networkReceivedBytes}
                    />
                )}
                {metrics.totalNetworkSentBytes != null && (
                    <ByteMetricTile
                        icon={<NorthIcon fontSize="small" />}
                        label="Sent"
                        value={metrics.totalNetworkSentBytes}
                        limit={limits?.networkSentBytes}
                    />
                )}
            </MetricGroup>

            <MetricGroup label="Disk">
                {metrics.totalDiskReadBytes != null && (
                    <ByteMetricTile
                        icon={<StorageIcon fontSize="small" />}
                        label="Read"
                        value={metrics.totalDiskReadBytes}
                        limit={limits?.diskReadBytes}
                    />
                )}
                {metrics.totalDiskWriteBytes != null && (
                    <ByteMetricTile
                        icon={<StorageIcon fontSize="small" />}
                        label="Write"
                        value={metrics.totalDiskWriteBytes}
                        limit={limits?.diskWriteBytes}
                    />
                )}
            </MetricGroup>
        </Box>
    );
}

export default JobResourceUsageMetrics;
