import RocketLaunchOutlinedIcon from '@mui/icons-material/RocketLaunchOutlined';
import VerifiedUserOutlinedIcon from '@mui/icons-material/VerifiedUserOutlined';
import { Box, Chip, Paper, Tooltip, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { WorkspaceIcon } from '../common/Icons';
import MiddleDot from '../common/MiddleDot';
import Pill from '../common/Pill';
import Timestamp from '../common/Timestamp';
import { MutationError } from '../common/error';
import { checkTypeLabel } from '../namespace/policies/policyDisplay';
import Link from '../routes/Link';
import { taskStagePath } from '../workspace/runs/runStageNavigation';
import RunTaskStageRunGateDecisionButtons from '../workspace/runs/taskstage/RunTaskStageRunGateDecisionButtons';
import ApprovalGateFindings from './ApprovalGateFindings';
import { ApprovalGateCardFragment_gate$key } from './__generated__/ApprovalGateCardFragment_gate.graphql';

interface Props {
    fragmentRef: ApprovalGateCardFragment_gate$key;
    // Connections the gate is dropped from once decided, so the card leaves the inbox.
    connectionIds: readonly string[];
    onDecided: () => void;
    onError: (error: MutationError) => void;
}

// ApprovalGateCard is one row of the approvals inbox: a run gate waiting on this caller, with enough
// of its policy check to decide without opening the run — which run it blocks and every failure line
// it reported, leaving the per-policy detail and approval progress to the policy stage page.
//
// The card itself is not a link. Its three ways out are links instead — the title to the blocked
// policy stage, the run ID to the run, the path to the workspace — so the card can hold controls and
// a disclosure without them having to cancel a navigation they sit inside.
function ApprovalGateCard({ fragmentRef, connectionIds, onDecided, onError }: Props) {
    const theme = useTheme();

    const gate = useFragment<ApprovalGateCardFragment_gate$key>(
        graphql`
        fragment ApprovalGateCardFragment_gate on RunGate
        {
            id
            metadata {
                createdAt
            }
            run {
                id
                isDestroy
                workspace {
                    fullPath
                }
            }
            # The check the gate governs. Its policies are what the approval is actually about.
            policyCheck {
                id
                checkType
                status
                stageName
                policies {
                    id
                    status
                }
                # The check's own preview of what its policies reported. This card is one of many in
                # a list, and a policy's full messages field is a read from object storage each —
                # which is exactly what this summary exists to avoid.
                messagesSummary {
                    messages
                    truncated
                }
            }
        }
      `, fragmentRef);

    const { run, policyCheck: check } = gate;

    // The title links to the stage the check belongs to, so it opens the view that is actually blocked
    // rather than the run's default tab.
    const workspacePath = `/groups/${run.workspace.fullPath}`;
    const runPath = `${workspacePath}/-/runs/${run.id}`;
    const checkPath = `${runPath}/${taskStagePath(check.stageName)}`;

    return (
        <Paper variant="outlined" component="article" sx={{ padding: 2 }}>
            {/* Header */}
            <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 2.5, flexWrap: 'wrap' }}>
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: '2px', minWidth: 0 }}>
                    {/* No status pill: every gate in this inbox is waiting on the caller, so the status is
                        the same on every card and says nothing. The title line carries when it was asked
                        for instead — no gap on the row, the dot supplies its own spacing. */}
                    <Box sx={{ display: 'flex', alignItems: 'center', flexWrap: 'wrap' }}>
                        <VerifiedUserOutlinedIcon
                            sx={{ width: 18, height: 18, mr: 0.875, flexShrink: 0 }}
                        />
                        <Typography variant="subtitle1" fontWeight={500}>
                            <Link color="inherit" to={checkPath}>
                                Policy check
                            </Link>
                        </Typography>
                        <Pill variant="outline" size="small" sx={{ ml: 1 }}>
                            {checkTypeLabel(check.checkType)}
                        </Pill>
                        <MiddleDot />
                        <Timestamp variant="body2" color="textSecondary" timestamp={gate.metadata.createdAt as string} />
                    </Box>
                    {/* Run context, as a second line of the title's own metadata: which run and which
                        workspace is what identifies this gate, so it belongs with the timestamp rather
                        than in a footer the reader reaches after the findings. */}
                    <Box sx={{ mt: 1, display: 'flex', alignItems: 'center', gap: '8px 16px', flexWrap: 'wrap', fontSize: theme.typography.body2.fontSize, color: theme.palette.text.secondary }}>
                        <Box sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.875 }}>
                            <RocketLaunchOutlinedIcon sx={{ width: 14, height: 14, color: theme.palette.text.disabled }} />
                            <Typography color="textSecondary" component="span" sx={{ fontSize: 'inherit', fontWeight: 500}}>
                                <Link color="inherit" to={runPath}>{run.id.substring(0, 8)}</Link>
                            </Typography>
                            {run.isDestroy && <Chip size="xs" label="Destroy" sx={{ color: 'runStatus.destroy' }} />}
                        </Box>
                        <Tooltip title={run.workspace.fullPath}>
                            <Box sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.875, minWidth: 0 }}>
                                {/* The app's workspace icon rather than a generic folder — a folder is
                                    what Icons.ts uses for a group, so it would name the wrong thing. */}
                                <WorkspaceIcon sx={{ width: 14, height: 14, color: theme.palette.text.disabled, flexShrink: 0 }} />
                                {/* The clipping stays on the wrapper, not the link: the anchor is inline
                                    content inside it, so a long path still ellipsizes rather than
                                    widening the row. */}
                                <Box component="span" sx={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                                    <Link sx={{ fontWeight: 500, color: theme.palette.text.secondary }} color="inherit" to={workspacePath}>{run.workspace.fullPath}</Link>
                                </Box>
                            </Box>
                        </Tooltip>
                    </Box>
                </Box>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.25, flexWrap: 'wrap' }}>
                    <RunTaskStageRunGateDecisionButtons
                        gateId={gate.id}
                        canOverride={false}
                        connectionIds={connectionIds}
                        onDecided={onDecided}
                        onError={onError}
                    />
                </Box>
            </Box>

            <ApprovalGateFindings
                messages={check.messagesSummary.messages}
                truncated={check.messagesSummary.truncated}
            />
        </Paper>
    );
}

export default ApprovalGateCard;
