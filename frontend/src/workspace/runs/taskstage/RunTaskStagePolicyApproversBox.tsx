import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import LockOutlinedIcon from '@mui/icons-material/LockOutlined';
import { Box, Collapse, Tooltip, Typography, useTheme } from '@mui/material';
import { alpha } from '@mui/material/styles';
import graphql from 'babel-plugin-relay/macro';
import { useState } from 'react';
import { useFragment } from 'react-relay/hooks';
import Pill from '../../../common/Pill';
import Link from '../../../routes/Link';
import { RunTaskStagePolicyApproversBoxFragment_gate$data, RunTaskStagePolicyApproversBoxFragment_gate$key } from './__generated__/RunTaskStagePolicyApproversBoxFragment_gate.graphql';
import { approvalsForRule, findRule } from './policyCheck';
import RunTaskStagePrincipalAvatar, { PRINCIPAL_AVATAR_SIZE } from './RunTaskStagePrincipalAvatar';

interface Props {
    gateRef: RunTaskStagePolicyApproversBoxFragment_gate$key;
    // The policy whose approval rule this box is for. A gate carries one rule per soft-failed policy,
    // and a rule's name is the policy's id.
    policyId: string;
}

type Approval = RunTaskStagePolicyApproversBoxFragment_gate$data['approvals'][number];

const APPROVE = 'APPROVE';

// A row per required approver: just the principal's name, linked to its details page when one is
// reachable. The type is not spelled out — the avatar carries it, a rounded square for a team and a
// circle for an individual.
//
// Teams show no approval state either: a decision records the user who made it and which rule it
// covers, never the team they approved as, so nothing team-scoped can be stated truthfully here.
// The header's avatar stack covers who actually approved the rule.
function ApproverRow({ avatar, name, to, state, stateColor }: {
    avatar: React.ReactNode,
    name: string,
    // When set, the name links here. Kept as a prop rather than letting callers pass a link as the
    // name so the row keeps deciding how an approver's name is styled.
    to?: string,
    state?: string,
    stateColor?: string,
}) {
    const theme = useTheme();

    return (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
            {avatar}
            <Box sx={{ flex: 1, minWidth: 0 }}>
                {/* color="inherit" so the link reads as the approver's name with a hover affordance,
                    not as an accent-colored control among the plain rows beside it. */}
                <Typography variant="subtitle2" component="div">
                    {to ? <Link color="inherit" to={to}>{name}</Link> : name}
                </Typography>
                {state && (
                    <Typography variant="caption" component="div" sx={{ color: stateColor ?? theme.palette.text.disabled }}>
                        {state}
                    </Typography>
                )}
            </Box>
        </Box>
    );
}

function RunTaskStagePolicyApproversBox({ gateRef, policyId }: Props) {
    const theme = useTheme();
    // Collapsed by default: the header summary is enough most of the time, and a policy card can
    // otherwise be dominated by the approver roster.
    const [expanded, setExpanded] = useState(false);

    const gate = useFragment<RunTaskStagePolicyApproversBoxFragment_gate$key>(
        graphql`
        fragment RunTaskStagePolicyApproversBoxFragment_gate on RunGate
        {
            approvalRules {
                name
                requiredApprovals
                allowedUsers {
                    id
                    username
                    email
                }
                allowedServiceAccounts {
                    id
                    name
                    resourcePath
                }
                allowedTeams {
                    id
                    name
                }
            }
            approvals {
                id
                decision
                createdBy
                coveredRules
                user {
                    id
                    email
                }
                serviceAccount {
                    id
                    name
                    resourcePath
                }
            }
        }
      `, gateRef);

    // A gate carries one rule per soft-failed policy, so a gate existing does not mean this box's
    // policy is gated — rendering nothing is the answer for a policy that passed, or whose failure is
    // not overridable by approval. Owning the lookup here spares the caller from repeating it.
    const rule = findRule(gate, policyId);
    if (!rule) {
        return null;
    }

    const approvals = approvalsForRule(gate, policyId);
    const approved = approvals.filter(a => a.decision === APPROVE);
    const met = approved.length >= rule.requiredApprovals;

    // The state shown on an individual's row. Only decisions we can attribute to a specific
    // principal are reported; everyone else is simply still pending.
    const stateFor = (matches: (approval: Approval) => boolean) => {
        const decision = approvals.find(matches);
        if (!decision) {
            return { state: 'Approval pending', stateColor: theme.palette.text.disabled };
        }
        return decision.decision === APPROVE
            ? { state: 'Approved', stateColor: theme.palette.success.main }
            : { state: 'Rejected', stateColor: theme.palette.error.main };
    };

    return (
        <Box
            sx={{
                mt: 1.5,
                background: alpha(theme.palette.common.white, 0.03),
                borderRadius: '6px'
            }}
        >
            {/* The header doubles as the disclosure toggle. It carries the whole summary — the
                approved-of-required count and who approved — so the collapsed state still answers
                "is this policy cleared?" without expanding. */}
            <Box
                onClick={() => setExpanded(open => !open)}
                aria-expanded={expanded}
                sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 1.5, flexWrap: 'wrap', cursor: 'pointer', p: 1.5 }}
            >
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <LockOutlinedIcon sx={{ width: 15, height: 15, color: theme.palette.warning.main }} />
                    <Typography variant="subtitle2" component="span">Required approvers</Typography>
                    <Pill
                        size="small"
                        color={met ? theme.palette.success.main : theme.palette.warning.main}
                    >
                        {approved.length} of {rule.requiredApprovals} approved
                    </Pill>
                </Box>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    {approved.length > 0 && (
                        <Box sx={{ display: 'flex' }}>
                            {approved.map((approval) => {
                                const label = approval.user?.email ?? approval.serviceAccount?.resourcePath ?? approval.createdBy;
                                return (
                                    <Tooltip key={approval.id} title={label}>
                                        <Box sx={{ ml: 1, boxShadow: `0 0 0 2px ${theme.palette.background.paper}`, borderRadius: '50%' }}>
                                            <RunTaskStagePrincipalAvatar
                                                kind={approval.user ? 'user' : 'serviceAccount'}
                                                size={PRINCIPAL_AVATAR_SIZE}
                                                label={label}
                                            />
                                        </Box>
                                    </Tooltip>
                                );
                            })}
                        </Box>
                    )}
                    <ExpandMoreIcon
                        sx={{
                            width: 18,
                            height: 18,
                            flexShrink: 0,
                            color: theme.palette.text.secondary,
                            transform: expanded ? 'rotate(180deg)' : 'none',
                            transition: '0.2s',
                        }}
                    />
                </Box>
            </Box>

            <Collapse in={expanded} unmountOnExit>
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5, p: 1.5 }}>
                    {rule.allowedTeams.map(team => (
                        <ApproverRow
                            key={team.id}
                            avatar={<RunTaskStagePrincipalAvatar kind="team" size={PRINCIPAL_AVATAR_SIZE} label={team.name} />}
                            name={team.name}
                            to={`/teams/${encodeURIComponent(team.name)}`}
                        />
                    ))}
                    {rule.allowedUsers.map(user => {
                        const { state, stateColor } = stateFor(a => a.user?.id === user.id);
                        return (
                            <ApproverRow
                                key={user.id}
                                avatar={<RunTaskStagePrincipalAvatar kind="user" size={PRINCIPAL_AVATAR_SIZE} label={user.email} />}
                                name={user.username}
                                state={state}
                                stateColor={stateColor}
                            />
                        );
                    })}
                    {rule.allowedServiceAccounts.map(sa => {
                        const { state, stateColor } = stateFor(a => a.serviceAccount?.id === sa.id);
                        return (
                            <ApproverRow
                                key={sa.id}
                                avatar={<RunTaskStagePrincipalAvatar kind="serviceAccount" size={PRINCIPAL_AVATAR_SIZE} label={sa.resourcePath} />}
                                name={sa.resourcePath}
                                state={state}
                                stateColor={stateColor}
                            />
                        );
                    })}
                    {rule.allowedTeams.length + rule.allowedUsers.length + rule.allowedServiceAccounts.length === 0 && (
                        <Typography variant="body2" color="textSecondary">
                            No approvers are currently resolvable
                        </Typography>
                    )}
                </Box>
            </Collapse>
        </Box>
    );
}

export default RunTaskStagePolicyApproversBox;
