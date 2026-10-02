import { Box, Chip, Paper, Stack, Typography } from '@mui/material';
import { alpha, useTheme } from '@mui/material/styles';
import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import CopyButton from '../common/CopyButton';
import Timestamp from '../common/Timestamp';
import Link from '../routes/Link';
import { WorkspaceDetailsStatusPanelFragment_workspace$key } from './__generated__/WorkspaceDetailsStatusPanelFragment_workspace.graphql';
import ModuleSourceLink from './ModuleSourceLink';
import RunStageIcons from './runs/RunStageIcons';

interface Props {
    fragmentRef: WorkspaceDetailsStatusPanelFragment_workspace$key
}

const shortId = (id: string) => `${id.substring(0, 8)}…`;

// Verb shown in the last-run header, e.g. "Run abc… applied"
const LAST_RUN_OUTCOME: Record<string, string> = {
    applied: 'applied',
    errored: 'errored',
    canceled: 'was canceled',
};

function SectionLabel({ children }: { children: React.ReactNode }) {
    return (
        <Typography variant="caption" color="textSecondary" sx={{ fontWeight: 500, letterSpacing: '.08em', textTransform: 'uppercase', lineHeight: '24px' }}>
            {children}
        </Typography>
    );
}

function VersionId({ id, to, toolTip }: { id: string, to: string, toolTip: string }) {
    return (
        <Stack direction="row" spacing={1} alignItems="center">
            <Link color="secondary" underline="hover" to={to} sx={{ fontFamily: 'monospace', fontSize: 16 }}>
                {shortId(id)}
            </Link>
            <CopyButton data={id} toolTip={toolTip} />
        </Stack>
    );
}

function DiffCount({ count, label, color }: { count: number, label: string, color: string }) {
    return (
        <Typography variant="body2" sx={{ color: count > 0 ? color : 'text.secondary' }}>
            {label}
        </Typography>
    );
}

function WorkspaceDetailsStatusPanel({ fragmentRef }: Props) {
    const theme = useTheme();

    const data = useFragment<WorkspaceDetailsStatusPanelFragment_workspace$key>(
        graphql`
      fragment WorkspaceDetailsStatusPanelFragment_workspace on Workspace
      {
        fullPath
        currentApplyRun {
            ...RunStageIconsFragment_run
            id
            isDestroy
        }
        currentStateVersion {
            id
            metadata {
                createdAt
            }
            inventory {
                resources {
                    __typename
                }
            }
            run {
                ...ModuleSourceLinkFragment_run
                ...RunStageIconsFragment_run
                id
                status
                apply {
                    status
                }
                plan {
                    summary {
                        resourceAdditions
                        resourceChanges
                        resourceDestructions
                    }
                }
                moduleSource
                moduleVersion
                configurationVersion {
                    id
                    metadata {
                        createdAt
                    }
                    vcsEvent {
                        id
                    }
                }
            }
        }
      }
    `, fragmentRef);

    const workspacePath = `/groups/${data.fullPath}/-`;
    const activeRun = data.currentApplyRun;
    const stateVersion = data.currentStateVersion;
    const sourceRun = stateVersion?.run;
    const resourceCount = stateVersion?.inventory.resources.length ?? 0;
    // Plan counts only reflect what was applied if the apply finished; a partial apply may differ
    const planSummary = sourceRun?.apply?.status === 'finished' ? sourceRun.plan?.summary : undefined;
    const hasSourceColumn =!!(sourceRun?.configurationVersion || sourceRun?.moduleSource);

    const lastRunColor = sourceRun
        ? theme.palette.runStatus[sourceRun.status as keyof typeof theme.palette.runStatus] ?? theme.palette.runStatus.unknown
        : undefined;

    const runColor = activeRun?.isDestroy ? theme.palette.error.main : theme.palette.info.main;

    return (
        <Paper sx={{ overflow: 'hidden' }}>
            {activeRun && <Box
                sx={{
                    display: 'flex',
                    alignItems: 'center',
                    flexWrap: 'wrap',
                    gap: 2,
                    px: 2.5,
                    py: 1.5,
                    bgcolor: alpha(runColor, 0.08),
                    borderBottom: stateVersion ? 1 : 0,
                    borderColor: 'divider',
                }}
            >
                <Box sx={{
                    width: 8,
                    height: 8,
                    borderRadius: '50%',
                    flexShrink: 0,
                    bgcolor: runColor,
                    '@keyframes activeRunPulse': {
                        '0%, 100%': { opacity: 1 },
                        '50%': { opacity: 0.35 },
                    },
                    animation: 'activeRunPulse 1.4s ease-in-out infinite',
                }} />
                <Typography component="div" sx={{ flex: 1, minWidth: 0 }}>
                    Run{' '}
                    <Link color="secondary" underline="hover" to={`${workspacePath}/runs/${activeRun.id}`} sx={{ fontFamily: 'monospace' }}>
                        {shortId(activeRun.id)}
                    </Link>
                    {' '}is in progress
                    {activeRun.isDestroy && <Chip
                        size="small"
                        variant="outlined"
                        color="error"
                        label="Destroy"
                        sx={{ ml: 1.5, fontWeight: 500, bgcolor: alpha(theme.palette.error.main, 0.08) }}
                    />}
                </Typography>
                <Box sx={{ width: { xs: '100%', sm: 200 } }}>
                    <RunStageIcons fragmentRef={activeRun} />
                </Box>
            </Box>}
            {stateVersion && <Box sx={{
                display: 'grid',
                gridTemplateColumns: { xs: '1fr', md: hasSourceColumn ? '1fr 1fr' : '1fr' },
            }}>
                <Stack spacing={1} sx={{ px: 2.5, py: 2 }}>
                    <SectionLabel>State Version</SectionLabel>
                    <Stack direction="row" spacing={1} alignItems="baseline">
                        <Typography sx={{ fontSize: 32, fontWeight: 600, lineHeight: 1 }}>
                            {resourceCount}
                        </Typography>
                        <Typography color="textSecondary">
                            {resourceCount === 1 ? 'resource' : 'resources'}
                        </Typography>
                    </Stack>
                    {planSummary && <Stack direction="row" spacing={1.5}>
                        <DiffCount count={planSummary.resourceAdditions} label={`+${planSummary.resourceAdditions} added`} color="planDiff.create" />
                        <DiffCount count={planSummary.resourceChanges} label={`~${planSummary.resourceChanges} changed`} color="planDiff.update" />
                        <DiffCount count={planSummary.resourceDestructions} label={`${planSummary.resourceDestructions} destroyed`} color="planDiff.delete" />
                    </Stack>}
                    {!sourceRun && <Typography variant="body2" color="textSecondary" component="div">
                        Updated <Timestamp timestamp={stateVersion.metadata.createdAt} /> by manual update
                    </Typography>}
                </Stack>
                {hasSourceColumn && <Stack spacing={1} sx={{
                    px: 2.5,
                    py: 2,
                    borderColor: 'divider',
                    borderTop: { xs: 1, md: 0 },
                    borderLeft: { xs: 0, md: 1 },
                }}>
                    {sourceRun?.configurationVersion && <React.Fragment>
                        <SectionLabel>Configuration Version</SectionLabel>
                        <VersionId
                            id={sourceRun.configurationVersion.id}
                            to={`${workspacePath}/configuration_versions/${sourceRun.configurationVersion.id}`}
                            toolTip="Copy configuration version ID"
                        />
                        <Typography variant="body2" color="textSecondary" component="div">
                            Uploaded <Timestamp timestamp={sourceRun.configurationVersion.metadata.createdAt} />
                            {sourceRun.configurationVersion.vcsEvent && ' · VCS'}
                        </Typography>
                    </React.Fragment>}
                    {!sourceRun?.configurationVersion && sourceRun?.moduleSource && <React.Fragment>
                        <SectionLabel>Module</SectionLabel>
                        <Box sx={{ fontSize: 16 }}>
                            <ModuleSourceLink fragmentRef={sourceRun} />
                        </Box>
                        <Stack direction="row" spacing={1} alignItems="center">
                            <Typography variant="body2" color="textSecondary">Version</Typography>
                            <Chip size="small" label={sourceRun.moduleVersion} />
                        </Stack>
                    </React.Fragment>}
                </Stack>}
            </Box>}
            {stateVersion && sourceRun && <Box
                sx={{
                    display: 'flex',
                    alignItems: 'center',
                    flexWrap: 'wrap',
                    gap: 2,
                    px: 2.5,
                    py: 1.5,
                    borderTop: 1,
                    borderColor: 'divider',
                }}
            >
                <Box sx={{ width: 8, height: 8, borderRadius: '50%', flexShrink: 0, bgcolor: lastRunColor }} />
                <Typography component="div" sx={{ flex: 1, minWidth: 0 }}>
                    Run{' '}
                    <Link color="secondary" underline="hover" to={`${workspacePath}/runs/${sourceRun.id}`} sx={{ fontFamily: 'monospace' }}>
                        {shortId(sourceRun.id)}
                    </Link>
                    {' '}{LAST_RUN_OUTCOME[sourceRun.status] ?? 'completed'}{' '}
                    <Typography component="span" color="textSecondary">
                        · <Timestamp timestamp={stateVersion.metadata.createdAt} />
                    </Typography>
                </Typography>
                <Box sx={{ width: { xs: '100%', sm: 200 } }}>
                    <RunStageIcons fragmentRef={sourceRun} />
                </Box>
            </Box>}
        </Paper>
    );
}

export default WorkspaceDetailsStatusPanel;
