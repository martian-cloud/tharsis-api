import { Box, Paper, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { MutationError } from '../../../common/error';
import RunStageStatusTypes from '../RunStageStatusTypes';
import RunTaskStageCancelRunButton from './RunTaskStageCancelRunButton';
import { stageLabel } from './policyCheck';
import { RunTaskStageStatusPanelFragment_taskStage$key } from './__generated__/RunTaskStageStatusPanelFragment_taskStage.graphql';

interface Props {
    runId: string;
    fragmentRef: RunTaskStageStatusPanelFragment_taskStage$key;
    onError: (error: MutationError) => void;
}

// The stage statuses from which the run can still be stopped: the stage has been admitted and has not
// reached a final state. A stage still in CREATED has not started, matching the plan stage, which only
// offers Cancel from pending/queued/running. AWAITING_OVERRIDE is included deliberately — a run parked
// on an approval it is never going to get is the case where cancelling matters most, and the server
// cancels the stage's checks the same way it would a running one.
const CANCELABLE_STAGE_STATUSES = ['PENDING', 'RUNNING', 'AWAITING_OVERRIDE'];

// getStageDescription returns the contextual line under the status. isPrePlan switches the wording for
// the stage that runs before the plan, which has no plan to evaluate yet. The statuses are
// RunTaskStageStatus values, so completed covers both "everything passed" and "a failure was
// overridden" — the check panel below distinguishes those.
function getStageDescription(status: string, isPrePlan: boolean): string | undefined {
    switch (status) {
        case 'CREATED':
            return isPrePlan ? 'The policy stage will run before the plan starts' : 'The policy stage will run after the plan completes';
        case 'PENDING':
            // A workspace-gated stage (pre-plan, pre-apply) waits here for the workspace slot, both on
            // its first run and when a check is retried after the run released the slot.
            return 'The policy stage is waiting for the workspace to become available';
        case 'RUNNING':
            return isPrePlan ? 'Policies are being evaluated against the run configuration' : 'Policies are being evaluated against the plan';
        case 'AWAITING_OVERRIDE':
            return 'A policy soft-failed and this run is waiting for the required approvals or an override before it can proceed';
        case 'COMPLETED':
            return 'The policy stage completed';
        case 'ERRORED':
            return 'The policy stage failed';
        case 'CANCELED':
            return 'The policy stage was canceled before it completed';
        case 'SKIPPED':
            return 'The policy stage was skipped';
        default:
            return undefined;
    }
}

// RunTaskStageStatusPanel is the run-stage summary card at the top of the policy stage view. It reports
// the stage's aggregate status, not any one check's, so it stays correct once a stage owns several
// checks — timing belongs to the individual check's job and is reported by RunTaskStagePolicyCheckPanel. The
// status colour and icon come from RunStageStatusTypes, which carries the stage-level COMPLETED and
// AWAITING_OVERRIDE entries alongside the run ones.
function RunTaskStageStatusPanel({ runId, fragmentRef, onError }: Props) {
    const theme = useTheme();

    const taskStage = useFragment<RunTaskStageStatusPanelFragment_taskStage$key>(
        graphql`
        fragment RunTaskStageStatusPanelFragment_taskStage on RunTaskStage
        {
            stageName
            status
            # A graceful cancel of a running check job is only a request until the runner confirms it,
            # so the Cancel action is replaced by the force-cancel alert once one is outstanding.
            policyChecks {
                currentJob {
                    cancelRequested
                }
            }
        }
      `, fragmentRef);

    const { status, stageName } = taskStage;
    const description = getStageDescription(status, stageName === 'PRE_PLAN');

    const cancelRequested = taskStage.policyChecks.some(check => check.currentJob?.cancelRequested);
    const cancelable = CANCELABLE_STAGE_STATUSES.includes(status) && !cancelRequested;

    const statusType = RunStageStatusTypes[status.toLowerCase()] ?? { label: 'unknown', color: 'runStatus.unknown', icon: () => null };
    const StatusIcon = statusType.icon;

    return (
        <Paper variant="outlined" component="section" sx={{ padding: 2 }}>
            <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 2 }}>
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.25 }}>
                    <Typography
                        variant="caption"
                        component="div"
                        sx={{
                            fontWeight: 700,
                            letterSpacing: '0.05em',
                            textTransform: 'uppercase',
                            color: theme.palette.text.disabled,
                        }}
                    >
                        Run Task Stage
                    </Typography>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.25 }}>
                        <StatusIcon sx={{ width: 24, height: 24, flexShrink: 0 }} />
                        <Typography variant="h6">
                            {stageLabel(stageName)} {statusType.label}
                        </Typography>
                    </Box>
                    {description && (
                        <Typography variant="body2" sx={{ color: theme.palette.text.secondary }}>
                            {description}
                        </Typography>
                    )}
                </Box>
                {cancelable && <RunTaskStageCancelRunButton runId={runId} onError={onError} />}
            </Box>
        </Paper>
    );
}

export default RunTaskStageStatusPanel;
