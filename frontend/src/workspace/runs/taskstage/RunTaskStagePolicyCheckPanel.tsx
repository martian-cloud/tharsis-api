import CancelOutlinedIcon from '@mui/icons-material/CancelOutlined';
import CheckIcon from '@mui/icons-material/Check';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import LockOpenOutlinedIcon from '@mui/icons-material/LockOpenOutlined';
import SubjectIcon from '@mui/icons-material/Subject';
import { Alert, Box, CircularProgress, Collapse, IconButton, Link as MuiLink, Paper, Typography, useTheme } from '@mui/material';
import { alpha } from '@mui/material/styles';
import graphql from 'babel-plugin-relay/macro';
import humanizeDuration from 'humanize-duration';
import moment from 'moment';
import { Suspense, useCallback, useMemo, useState } from 'react';
import { useFragment } from 'react-relay/hooks';
import { useSearchParams } from 'react-router-dom';
import Pill, { resolvePaletteColor } from '../../../common/Pill';
import Timestamp from '../../../common/Timestamp';
import { MutationError } from '../../../common/error';
import { checkTypeLabel } from '../../../namespace/policies/policyDisplay';
import JobLogs from '../JobLogs';
import RunStageStatusTypes from '../RunStageStatusTypes';
import RunTaskStageOverrideProgressBox from './RunTaskStageOverrideProgressBox';
import RunTaskStageOverrideRunButton from './RunTaskStageOverrideRunButton';
import RunTaskStagePolicyCheckPreviousJobsMenu from './RunTaskStagePolicyCheckPreviousJobsMenu';
import RunTaskStagePolicyErrorsBox from './RunTaskStagePolicyErrorsBox';
import RunTaskStagePrincipalAvatar, { PRINCIPAL_AVATAR_SIZE } from './RunTaskStagePrincipalAvatar';
import RunTaskStageRetryPolicyCheckButton from './RunTaskStageRetryPolicyCheckButton';
import RunTaskStageRunGateDecisionButtons from './RunTaskStageRunGateDecisionButtons';
import RunTaskStageSectionLabel from './RunTaskStageSectionLabel';
import { RunTaskStagePolicyCheckPanelFragment_check$key } from './__generated__/RunTaskStagePolicyCheckPanelFragment_check.graphql';
import { POLICY_FAILED, gateProgress, isPolicyCheckFinal } from './policyCheck';

// Statuses a check can be retried from; the API rejects every other state. A soft-failed check is
// retryable alongside the two failure states because re-evaluating is the right resolution once the
// policy itself has been fixed — the alternative is overriding a failure that no longer applies.
const RETRYABLE = ['ERRORED', 'CANCELED', 'SOFT_FAILED'];
const LOGS_HEIGHT = 420;

interface Props {
    // The check's own fragment carries no run id, and the mutations in the header need one.
    runId: string;
    fragmentRef: RunTaskStagePolicyCheckPanelFragment_check$key;
    onError: (error: MutationError) => void;
}

function Divided({ children, gap }: { children: React.ReactNode, gap?: number }) {
    const theme = useTheme();
    return (
        <Box
            sx={{
                mt: 2,
                pt: 2,
                borderTop: `1px solid ${theme.palette.divider}`,
                display: 'flex',
                flexDirection: 'column',
                gap: gap ?? 1.5,
            }}
        >
            {children}
        </Box>
    );
}

function RunTaskStagePolicyCheckPanel({ runId, fragmentRef, onError }: Props) {
    const theme = useTheme();
    const [searchParams, setSearchParams] = useSearchParams();

    const check = useFragment<RunTaskStagePolicyCheckPanelFragment_check$key>(
        graphql`
        fragment RunTaskStagePolicyCheckPanelFragment_check on PolicyCheck
        {
            checkType
            status
            nodePath
            currentJob {
                id
                timestamps {
                    runningAt
                    finishedAt
                }
            }
            # Count only — the previous jobs themselves are fetched by the popover that lists them.
            jobs(first: 0) {
                totalCount
            }
            policies {
                id
                status
                enforcementLevel
            }
            # The check's preview of what its policies reported, rather than each policy's full
            # messages field — those are read from object storage one policy at a time, and the cards
            # further down the page already select them.
            messagesSummary {
                messages
                truncated
            }
            runGate {
                id
                status
                # Set only when the gate was overridden rather than approved. updatedAt doubles as the
                # time it was cleared — both approved and overridden are terminal for a gate.
                overriddenBy
                overrideComment
                metadata {
                    updatedAt
                }
                approvalRules {
                    name
                    requiredApprovals
                }
                approvals {
                    id
                    decision
                    comment
                    createdBy
                    coveredRules
                    metadata {
                        createdAt
                    }
                    user {
                        id
                        username
                        email
                    }
                    serviceAccount {
                        id
                        name
                        resourcePath
                    }
                }
            }
        }
      `, fragmentRef);

    const { checkType, status, nodePath, policies, runGate: gate } = check;
    const currentJobId = check.currentJob?.id;

    // Open by default while the check is still producing output, so an in-progress evaluation is
    // visible without an extra click. A lazy initializer, not a status-driven effect: once the panel
    // has rendered, a later status change (e.g. the check finishing) must not yank open logs the user
    // deliberately closed, or re-close ones they're reading.
    const [logsOpen, setLogsOpen] = useState(() => !isPolicyCheckFinal(status));

    // With no jobId param the logs follow the check's latest job. With one set (picked from the
    // previous jobs popover) the view stays pinned to that job until the param is cleared — even as
    // a retry produces a newer job. The param is shared with the plan and apply stages, which pin
    // their logs the same way.
    const pinnedJobId = searchParams.get('jobId');
    const effectiveJobId = pinnedJobId ?? currentJobId;
    const viewingEarlierJob = !!pinnedJobId && !!currentJobId && pinnedJobId !== currentJobId;

    // The pin is merged into the existing params rather than replacing the query string, so it does
    // not drop the ?line= deep link the log viewer sets.
    const setPinnedJob = useCallback((jobId?: string) => {
        setSearchParams(prev => {
            const next = new URLSearchParams(prev);
            if (jobId) {
                next.set('jobId', jobId);
            } else {
                next.delete('jobId');
            }
            return next;
        }, { replace: true });
    }, [setSearchParams]);

    // Jobs before the current one. totalCount counts every attempt, including the current job.
    const previousJobCount = Math.max(check.jobs.totalCount - (currentJobId ? 1 : 0), 0);

    // How long the evaluation itself took. Null until the job has both started and finished, so a
    // running check shows nothing rather than a duration that keeps growing.
    const durationMs = useMemo(() => {
        const timestamps = check.currentJob?.timestamps;
        return timestamps?.runningAt && timestamps?.finishedAt
            ? moment(timestamps.finishedAt as moment.MomentInput).diff(moment(timestamps.runningAt as moment.MomentInput))
            : null;
    }, [check.currentJob]);

    const failed = policies.filter(p => p.status === POLICY_FAILED);

    // A check whose only failures are advisory still reports PASSED — advisory failures never gate
    // or block anything — but showing a plain "Passed" pill next to the failed-policy count below it
    // reads as contradictory. Called out in its own warning-toned status instead, distinct from a
    // clean pass with nothing to flag.
    const hasAdvisoryFailures = failed.some(p => p.enforcementLevel === 'ADVISORY');
    const statusType = status === 'PASSED' && hasAdvisoryFailures
        ? { label: 'Passed with advisories', color: 'runStatus.awaiting_decision' }
        : RunStageStatusTypes[status.toLowerCase()] ?? { label: 'unknown', color: 'runStatus.unknown' };
    const softFailed = status === 'SOFT_FAILED';
    const retryable = RETRYABLE.includes(status);

    const pendingGate = gate?.status === 'PENDING' ? gate : undefined;

    // A gate whose required approvals were bypassed outright. Distinct from one that reached
    // APPROVED by collecting them — that one leaves overriddenBy null and shows up under Reviews
    // instead. Both leave the check OVERRIDDEN, so the gate is the only thing that tells them apart.
    const overriddenGate = gate?.status === 'OVERRIDDEN' && gate.overriddenBy ? gate : undefined;

    // Only a gate that carried approval rules had approvals to bypass. A rule-less gate is the one
    // nobody was eligible to approve — its failed policies declared no approvers — so an override is the
    // only way it can ever clear, and reporting that as "bypassed" would name a requirement that never
    // existed.
    const bypassedApprovals = !!overriddenGate && overriddenGate.approvalRules.length > 0;

    const progress = gateProgress(gate, failed);

    const decisions = gate?.approvals ?? [];

    return (
        <Paper variant="outlined" component="section" sx={{ padding: 2 }}>
            {/* Header */}
            <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 2.5, flexWrap: 'wrap' }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, minWidth: 0 }}>
                    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '2px', minWidth: 0 }}>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.25, flexWrap: 'wrap' }}>
                            <Typography variant="subtitle1">
                                {checkTypeLabel(checkType)} Policy Results
                            </Typography>
                            <Pill color={resolvePaletteColor(theme, statusType.color)}>
                                {statusType.label}
                            </Pill>
                        </Box>
                        {/* The evaluated-count summary doesn't depend on a job existing; the duration
                            half does, and is simply absent for a check with no job (an in-API check,
                            or one that hasn't reported yet). */}
                        {policies.length > 0 && <Typography variant="body2" sx={{ color: theme.palette.text.disabled }}>
                            Evaluated {policies.length} polic{policies.length === 1 ? 'y' : 'ies'}
                            {durationMs !== null && ` · took ${humanizeDuration(durationMs)}`}
                        </Typography>}
                    </Box>
                </Box>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.25, flexWrap: 'wrap' }}>
                    {/* A pending gate with no approval rules has nobody eligible to approve it, so
                        Override stands alone rather than hanging off an Approve it cannot reach. */}
                    {pendingGate && pendingGate.approvalRules.length === 0 && (
                        <RunTaskStageOverrideRunButton gateId={pendingGate.id} onError={onError} />
                    )}
                    {pendingGate && pendingGate.approvalRules.length > 0 && (
                        <RunTaskStageRunGateDecisionButtons
                            gateId={pendingGate.id}
                            canOverride={softFailed}
                            onError={onError}
                        />
                    )}
                </Box>
            </Box>

            {/* What the check is waiting for, next to the buttons that resolve it. Only a soft-failed
                check is waiting on approvals; once it clears, the outcome is reported by the gate's
                status and the Reviews list below instead. */}
            {softFailed && <RunTaskStageOverrideProgressBox gatedCount={progress.gated} approvedCount={progress.approved} />}

            {/* Counts. A verdict of zero says nothing, so each side shows only when it has something
                to report — and the divider goes too when the check has yet to evaluate anything. */}
            {(failed.length > 0) && (
                <Divided gap={0}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 2.5, flexWrap: 'wrap' }}>
                        <Box sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.75, fontSize: theme.typography.body2.fontSize, fontWeight: 600, color: theme.palette.error.main }}>
                            <CancelOutlinedIcon sx={{ width: 16, height: 16 }} />
                            {failed.length} policy failed
                        </Box>
                    </Box>
                </Divided>
            )}

            {/* The whole picture without scrolling the per-policy cards below. */}
            <RunTaskStagePolicyErrorsBox
                messages={check.messagesSummary.messages}
                truncated={check.messagesSummary.truncated}
            />

            {/* The override that cleared the gate, titled for what it actually skipped: approvals the
                gate required, or nothing but the failure itself when it had no approval rules. Mutually
                exclusive with the progress box in the header: an overridden gate means the check is no
                longer soft-failed. */}
            {overriddenGate && (
                <Box
                    sx={{
                        mt: 2,
                        background: alpha(theme.palette.warning.main, 0.09),
                        border: `1px solid ${alpha(theme.palette.warning.main, 0.25)}`,
                        borderRadius: '6px',
                        padding: '14px 16px',
                        display: 'flex',
                        alignItems: 'flex-start',
                        gap: 1.5,
                    }}
                >
                    <LockOpenOutlinedIcon sx={{ width: 18, height: 18, color: theme.palette.warning.main, flexShrink: 0, mt: '2px' }} />
                    <Box sx={{ minWidth: 0, display: 'flex', flexDirection: 'column', gap: 0.5 }}>
                        <Typography variant="subtitle2" component="div">
                            {bypassedApprovals ? 'Approvals bypassed' : 'Policy overridden'}
                        </Typography>
                        <Typography variant="body2" sx={{ color: theme.palette.text.secondary }}>
                            Overridden by {overriddenGate.overriddenBy}
                            {' · '}
                            <Timestamp timestamp={overriddenGate.metadata.updatedAt as string} />
                        </Typography>
                        {overriddenGate.overrideComment && (
                            <Typography
                                variant="body2"
                                sx={{
                                    mt: 0.25,
                                    color: theme.palette.text.secondary,
                                    borderLeft: `2px solid ${alpha(theme.palette.common.white, 0.23)}`,
                                    paddingLeft: '10px',
                                    whiteSpace: 'pre-wrap',
                                }}
                            >
                                {overriddenGate.overrideComment}
                            </Typography>
                        )}
                    </Box>
                </Box>
            )}

            {/* Reviews */}
            {decisions.length > 0 && (
                <Divided>
                    <RunTaskStageSectionLabel>Reviews</RunTaskStageSectionLabel>
                    {decisions.map(approval => {
                        const label = approval.user?.email ?? approval.serviceAccount?.resourcePath ?? approval.createdBy;
                        const approvedDecision = approval.decision === 'APPROVE';
                        return (
                            <Box key={approval.id} sx={{ display: 'flex', gap: 1.25, alignItems: 'flex-start' }}>
                                <RunTaskStagePrincipalAvatar
                                    kind={approval.user ? 'user' : 'serviceAccount'}
                                    size={PRINCIPAL_AVATAR_SIZE}
                                    label={label}
                                />
                                <Box sx={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', gap: 0.5 }}>
                                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap' }}>
                                        <Typography variant="subtitle2" component="span">{label}</Typography>
                                        <Box
                                            sx={{
                                                display: 'inline-flex',
                                                alignItems: 'center',
                                                gap: '4px',
                                                fontSize: theme.typography.caption.fontSize,
                                                fontWeight: 600,
                                                color: approvedDecision ? theme.palette.success.main : theme.palette.error.main,
                                            }}
                                        >
                                            {approvedDecision
                                                ? <CheckIcon sx={{ width: 13, height: 13 }} />
                                                : <CancelOutlinedIcon sx={{ width: 13, height: 13 }} />}
                                            {approvedDecision ? 'Approved' : 'Rejected'}
                                        </Box>
                                        <Typography variant="caption" sx={{ color: theme.palette.text.disabled }}>
                                            <Timestamp timestamp={approval.metadata.createdAt as string} />
                                        </Typography>
                                    </Box>
                                    {approval.comment && (
                                        <Typography
                                            variant="body2"
                                            sx={{
                                                m: 0,
                                                color: theme.palette.text.secondary,
                                                borderLeft: `2px solid ${alpha(theme.palette.common.white, 0.23)}`,
                                                paddingLeft: '10px',
                                            }}
                                        >
                                            {approval.comment}
                                        </Typography>
                                    )}
                                </Box>
                            </Box>
                        );
                    })}
                </Divided>
            )}

            {/* Job logs disclosure. A check with no job — one that evaluates in-API rather than on a
                runner (e.g. module attestation), or one that simply hasn't reported yet — has no
                logs, no previous-jobs history, and no duration to show. It also has nothing to
                retry: retry re-runs the job, and a check with no job has none to re-run. So the
                whole disclosure, retry included, is omitted rather than rendered empty. */}
            {currentJobId && (
                <Box sx={{ mt: 2, pt: 1.75, borderTop: `1px solid ${theme.palette.divider}` }}>
                    {/* The toggle label and the chevron are separate controls so the actions
                        between them are not nested inside a button. */}
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                        <Box
                            component="button"
                            onClick={() => setLogsOpen(open => !open)}
                            sx={{
                                display: 'flex',
                                alignItems: 'center',
                                gap: 0.75,
                                background: 'none',
                                border: 'none',
                                padding: 0,
                                cursor: 'pointer',
                                fontSize: theme.typography.body2.fontSize,
                                fontFamily: 'inherit',
                                color: theme.palette.secondary.main,
                                '&:hover': { textDecoration: 'underline' },
                            }}
                        >
                            <SubjectIcon sx={{ width: 15, height: 15 }} />
                            {logsOpen ? 'Hide job logs' : 'View job logs'}
                        </Box>
                        <Box sx={{ flex: 1 }} />
                        {/* Actions that only make sense against logs on screen: picking an
                            earlier job to read, and re-evaluating a check whose logs say why it
                            needs re-evaluating. Retry is deliberately kept out of the header
                            actions at the top of the panel, which decide the check as it stands
                            (approve, reject, override) — a retry throws that evaluation away. */}
                        {logsOpen && (
                            <>
                                <RunTaskStagePolicyCheckPreviousJobsMenu
                                    runId={runId}
                                    nodePath={nodePath}
                                    currentJobId={currentJobId}
                                    previousJobCount={previousJobCount}
                                    selectedJobId={pinnedJobId}
                                    onSelectJob={setPinnedJob}
                                />
                                {retryable && (
                                    <RunTaskStageRetryPolicyCheckButton runId={runId} nodePath={nodePath} onError={onError} />
                                )}
                            </>
                        )}
                        <IconButton
                            size="small"
                            aria-label={logsOpen ? 'Collapse job logs' : 'Expand job logs'}
                            onClick={() => setLogsOpen(open => !open)}
                        >
                            <ExpandMoreIcon
                                sx={{ width: 18, height: 18, transform: logsOpen ? 'rotate(180deg)' : 'none', transition: '0.2s' }}
                            />
                        </IconButton>
                    </Box>
                    {/* Mounted only once opened so the logs query isn't fired on page load. */}
                    <Collapse in={logsOpen} unmountOnExit>
                        <Box sx={{ mt: 1.5, display: 'flex', flexDirection: 'column', gap: 1.5 }}>
                            {logsOpen && effectiveJobId && (
                                <>
                                    {viewingEarlierJob && <Alert severity="warning">
                                        Showing logs for an earlier job{' '}
                                        <MuiLink
                                            component="button"
                                            color="inherit"
                                            underline="always"
                                            onClick={() => setPinnedJob(undefined)}
                                            sx={{ fontFamily: 'inherit', fontSize: 'inherit', verticalAlign: 'baseline' }}
                                        >
                                            (view latest job)
                                        </MuiLink>
                                    </Alert>}
                                    <Suspense fallback={
                                        <Box sx={{ minHeight: 120, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                                            <CircularProgress />
                                        </Box>
                                    }>
                                        <JobLogs jobId={effectiveJobId} scrollMode="container" height={LOGS_HEIGHT} />
                                    </Suspense>
                                </>
                            )}
                        </Box>
                    </Collapse>
                </Box>
            )}
        </Paper>
    );
}

export default RunTaskStagePolicyCheckPanel;
