import RefreshIcon from '@mui/icons-material/Refresh';
import { LoadingButton } from '@mui/lab';
import graphql from 'babel-plugin-relay/macro';
import { useMutation } from 'react-relay/hooks';
import { MutationError } from '../../../common/error';
import { RunTaskStageRetryPolicyCheckButtonMutation } from './__generated__/RunTaskStageRetryPolicyCheckButtonMutation.graphql';

interface Props {
    runId: string;
    // The check's node path (e.g. "post_plan.opa"), taken from PolicyCheck.nodePath. It cannot be
    // rebuilt from stageName/checkType, which the API returns upper-cased.
    nodePath: string;
    onError: (error: MutationError) => void;
}

// RunTaskStageRetryPolicyCheckButton re-runs a failed, canceled, or soft-failed policy check. The API rejects a
// retry in any other state, so callers should render this only for those three. Retrying a soft-failed
// check discards its gate and any approvals recorded on it, which is deliberate — the approvals were
// against a verdict that is being replaced — and is not confirmed here.
function RunTaskStageRetryPolicyCheckButton({ runId, nodePath, onError }: Props) {
    const [commitRetryRunNode, commitRetryRunNodeInFlight] = useMutation<RunTaskStageRetryPolicyCheckButtonMutation>(graphql`
        mutation RunTaskStageRetryPolicyCheckButtonMutation($input: RetryRunNodeInput!) {
            retryRunNode(input: $input) {
                run {
                    ...RunDetailsRunTaskStageFragment_taskStage
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const retry = () => {
        commitRetryRunNode({
            variables: { input: { runId, nodePath } },
            onCompleted: data => {
                if (data.retryRunNode.problems.length) {
                    onError({
                        severity: 'warning',
                        message: data.retryRunNode.problems.map(problem => problem.message).join('; ')
                    });
                }
            },
            onError: error => {
                onError({
                    severity: 'error',
                    message: `Unexpected Error Occurred: ${error.message}`
                });
            }
        });
    };

    return (
        <LoadingButton
            color="secondary"
            loading={commitRetryRunNodeInFlight}
            size="small"
            variant="outlined"
            startIcon={<RefreshIcon />}
            onClick={retry}
        >
            Retry Job
        </LoadingButton>
    );
}

export default RunTaskStageRetryPolicyCheckButton;
