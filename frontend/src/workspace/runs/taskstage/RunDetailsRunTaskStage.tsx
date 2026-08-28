import { Box, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { MutationError } from '../../../common/error';
import ForceCancelRunAlert from '../ForceCancelRunAlert';
import NoRunnerAlert from '../NoRunnerAlert';
import RunDetailsStageHeader from '../RunDetailsStageHeader';
import RunTaskStagePolicyCheckPanel from './RunTaskStagePolicyCheckPanel';
import RunTaskStagePolicyCheckPolicyCard from './RunTaskStagePolicyCheckPolicyCard';
import RunTaskStageStatusPanel from './RunTaskStageStatusPanel';
import { RunDetailsRunTaskStageFragment_taskStage$key } from './__generated__/RunDetailsRunTaskStageFragment_taskStage.graphql';
import { stageLabel } from './policyCheck';

interface Props {
    // Which policy stage this route renders. Supplied by the route rather than read from the URL,
    // so there is no stage to default to and no ambiguity about which stage is on screen.
    stageName: 'PRE_PLAN' | 'POST_PLAN' | 'PRE_APPLY' | 'POST_APPLY'
    fragmentRef: RunDetailsRunTaskStageFragment_taskStage$key
    onError: (error: MutationError) => void
}

function RunDetailsRunTaskStage(props: Props) {
    const data = useFragment<RunDetailsRunTaskStageFragment_taskStage$key>(
        graphql`
        fragment RunDetailsRunTaskStageFragment_taskStage on Run
        {
            id
            createdBy
            metadata {
                createdAt
            }
            # Once a cancel has been requested but not yet confirmed by the runner, the run offers a
            # force cancel instead — the same escalation path the plan and apply stages use.
            ...ForceCancelRunAlertFragment_run
            taskStages {
                stageName
                status
                ...RunTaskStageStatusPanelFragment_taskStage
                policyChecks {
                    currentJob {
                        cancelRequested
                        ...NoRunnerAlertFragment_job
                    }
                    # The gate blocking this check, or null when the check is not blocked. A gate
                    # belongs to exactly one check, so a run that soft-fails at more than one stage
                    # keeps each stage's approvers separate. approvalRules is empty when the failed
                    # policies declared no approvers — that gate is cleared by override alone.
                    runGate {
                        approvalRules {
                            name
                        }
                        ...RunTaskStagePolicyCheckPolicyCardFragment_gate
                    }
                    policies {
                        id
                        status
                        ...RunTaskStagePolicyCheckPolicyCardFragment_policy
                    }
                    ...RunTaskStagePolicyCheckPanelFragment_check
                }
            }
        }
      `, props.fragmentRef);

    const taskStage = data.taskStages.find(s => s.stageName === props.stageName);
    const check = taskStage?.policyChecks[0];

    if (!taskStage) {
        // Runs created before the policy feature (or with no attached policies) have no
        // policy stage; the sidebar never links here, so this only renders on direct URL
        // navigation.
        return (
            <Box sx={{ marginTop: 4 }} display="flex" justifyContent="center">
                <Typography color="textSecondary">
                    This run does not have a {stageLabel(props.stageName)} policy stage
                </Typography>
            </Box>
        );
    }

    if (!check) {
        // The policy stage exists but has no policy checks to display.
        return (
            <Box sx={{ marginTop: 4 }} display="flex" justifyContent="center">
                <Typography color="textSecondary">This policy stage has no checks</Typography>
            </Box>
        );
    }

    const policies = check.policies;

    return (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            {check.currentJob?.cancelRequested && taskStage.status !== 'CANCELED' && <ForceCancelRunAlert fragmentRef={data} />}
            {check.currentJob && <NoRunnerAlert fragmentRef={check.currentJob} />}
            <RunDetailsStageHeader
                stage={stageLabel(taskStage.stageName)}
                triggeredAt={data.metadata.createdAt as string}
                triggeredBy={data.createdBy}
            />

            <RunTaskStageStatusPanel runId={data.id} fragmentRef={taskStage} onError={props.onError} />

            <RunTaskStagePolicyCheckPanel
                runId={data.id}
                fragmentRef={check}
                onError={props.onError}
            />

            <Box component="section" sx={{ display: 'flex', flexDirection: 'column', gap: 1.5, mt: 2 }}>
                <Typography variant="h6" component="h2" sx={{ fontWeight: 400, m: 0 }}>Policies</Typography>
                {policies.length === 0 && (
                    <Typography variant="body2" color="textSecondary">
                        No policies were evaluated for this run
                    </Typography>
                )}
                {policies.map(policy => (
                    <RunTaskStagePolicyCheckPolicyCard key={policy.id} policyRef={policy} gateRef={check.runGate} />
                ))}
            </Box>
        </Box>
    );
}

export default RunDetailsRunTaskStage;
