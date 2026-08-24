import CancelOutlinedIcon from '@mui/icons-material/CancelOutlined';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import ChevronRightIcon from '@mui/icons-material/ChevronRight';
import FolderOutlinedIcon from '@mui/icons-material/FolderOutlined';
import { Box, Paper, Tooltip, Typography, useTheme } from '@mui/material';
import { darken } from '@mui/material/styles';
import graphql from 'babel-plugin-relay/macro';
import { ReactNode } from 'react';
import { useFragment } from 'react-relay/hooks';
import { Link as RouterLink } from 'react-router-dom';
import RunTaskStagePolicyApproversBox from './RunTaskStagePolicyApproversBox';
import Pill from '../../../common/Pill';
import { RunTaskStagePolicyCheckPolicyCardFragment_gate$key } from './__generated__/RunTaskStagePolicyCheckPolicyCardFragment_gate.graphql';
import { RunTaskStagePolicyCheckPolicyCardFragment_policy$key } from './__generated__/RunTaskStagePolicyCheckPolicyCardFragment_policy.graphql';
import { POLICY_FAILED, POLICY_PASSED } from './policyCheck';

interface Props {
    policyRef: RunTaskStagePolicyCheckPolicyCardFragment_policy$key;
    // The gate governing the whole check, null when it has none. The approvers box below picks out the
    // rule for this card's policy, if there is one.
    gateRef: RunTaskStagePolicyCheckPolicyCardFragment_gate$key | null | undefined;
}

// One labelled cell of the card's details grid. The micro-label is deliberately not RunTaskStageSectionLabel —
// that is the larger heading used above a whole block ("Reviews", "All policy errors"), and these sit
// inside a row of cells where it would compete with the values it labels.
function DetailCell({ label, children }: { label: string, children: ReactNode }) {
    const theme = useTheme();
    return (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: '2px', minWidth: 0 }}>
            <Typography
                variant="caption"
                sx={{
                    fontWeight: 600,
                    letterSpacing: '0.08em',
                    textTransform: 'uppercase',
                    color: theme.palette.text.disabled,
                }}
            >
                {label}
            </Typography>
            {children}
        </Box>
    );
}

// The value half of a DetailCell. Monospace because every one of them is an identifier or a version
// the reader may need to compare against something they have elsewhere.
function DetailValue({ children }: { children: ReactNode }) {
    return (
        <Typography variant="code" sx={{ overflowWrap: 'anywhere' }}>
            {children}
        </Typography>
    );
}

function RunTaskStagePolicyCheckPolicyCard({ policyRef, gateRef }: Props) {
    const theme = useTheme();

    const policy = useFragment<RunTaskStagePolicyCheckPolicyCardFragment_policy$key>(
        graphql`
        fragment RunTaskStagePolicyCheckPolicyCardFragment_policy on PolicyCheckPolicy
        {
            id
            # name and description are snapshotted at run creation, so they describe the policy as it
            # was evaluated. Both are empty on checks created before the fields existed.
            name
            description
            packageSource
            # The constraint as snapshotted, not a resolved version — empty means latest.
            packageVersionConstraint
            enforcementLevel
            status
            # One entry per violation. Read from object storage per policy, which is why the panel
            # above uses the check's messagesSummary instead of collecting these.
            messages
            provenance {
                policyTrn
            }
            # Only needed to build the "View policy details" link — the snapshot above is what the card
            # shows. Null once the policy is deleted, which is why the link is optional.
            policy {
                id
                groupPath
            }
        }
      `, policyRef);

    // Nothing here is read directly — the card only needs to hand the gate to the approvers box, which
    // decides whether this policy is gated at all.
    const gate = useFragment<RunTaskStagePolicyCheckPolicyCardFragment_gate$key>(
        graphql`
        fragment RunTaskStagePolicyCheckPolicyCardFragment_gate on RunGate
        {
            ...RunTaskStagePolicyApproversBoxFragment_gate
        }
      `, gateRef);

    const failed = policy.status === POLICY_FAILED;
    const passed = policy.status === POLICY_PASSED;
    // An unevaluated policy (status is '' until the check reports) gets no verdict pill at all rather
    // than being shown as passing.
    const accent = failed ? theme.palette.error.main : theme.palette.success.main;

    return (
        <Paper
            variant="outlined"
            component="article"
            sx={{
                // A step below the panels above rather than level with them: these cards are a list
                // of details under those panels, not peers of them. Lighter than the page all the
                // same, so the gap between two cards reads as a channel and they separate from each
                // other — which a fill level with the page cannot do, whichever direction it moves.
                background: darken(theme.palette.background.paper, 0.20),
                padding: 2,
                display: 'flex',
                flexDirection: 'column',
                gap: 2,
            }}
        >
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.75, minWidth: 0 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, flexWrap: 'wrap' }}>
                    {/* The snapshot's name, falling back to the package for checks created before the
                        name was snapshotted. */}
                    <Typography variant="subtitle1" sx={{ fontWeight: 700 }}>
                        {policy.name || policy.packageSource}
                    </Typography>
                    <Pill variant="outline">
                        {policy.enforcementLevel.replace(/_/g, ' ').toLowerCase()}
                    </Pill>
                    {(failed || passed) && (
                        // ml auto rather than a spacer element, so the verdict stays right-aligned
                        // even once the row wraps on a narrow viewport.
                        <Pill
                            variant="solid"
                            color={accent}
                            sx={{ ml: 'auto' }}
                            icon={failed
                                ? <CancelOutlinedIcon sx={{ width: 14, height: 14 }} />
                                : <CheckCircleOutlineIcon sx={{ width: 14, height: 14 }} />}
                        >
                            {failed ? 'Failed' : 'Passed'}
                        </Pill>
                    )}
                </Box>
                {policy.description && (
                    <Typography variant="body2" sx={{ color: theme.palette.text.secondary }}>
                        {policy.description}
                    </Typography>
                )}
            </Box>

            {/* What was actually evaluated. Collapses to a single column when the card is too narrow
                to keep three readable. */}
            <Box
                sx={{
                    display: 'grid',
                    gridTemplateColumns: { xs: '1fr', sm: '0.7fr 1.5fr 0.8fr' },
                    gap: '16px 24px',
                }}
            >
                <DetailCell label="Policy type">
                    <DetailValue>OPA</DetailValue>
                </DetailCell>
                <DetailCell label="Package source">
                    <DetailValue>{policy.packageSource}</DetailValue>
                </DetailCell>
                <DetailCell label="Package version">
                    {/* Says outright that this is a constraint: the check resolves it when it runs, so
                        the version that was really evaluated is only in the job log. */}
                    <Tooltip title="The version constraint this policy was pinned to. The concrete version evaluated is recorded in the policy check's job log.">
                        <span>
                            <DetailValue>{policy.packageVersionConstraint || 'latest'}</DetailValue>
                        </span>
                    </Tooltip>
                </DetailCell>
            </Box>

            {gate && <RunTaskStagePolicyApproversBox gateRef={gate} policyId={policy.id} />}

            {/* Kept quiet: the verdict pill above already says the policy failed, so the messages only
                have to be readable, not alarming. The check's panel above collects the same messages
                across every policy in a form that does draw the eye. */}
            {failed && policy.messages.length > 0 && (
                <Box
                    sx={{
                        borderTop: `1px solid ${theme.palette.divider}`,
                        pt: 1.5,
                        display: 'flex',
                        flexDirection: 'column',
                        gap: 0.75,
                    }}
                >
                    <Typography variant="caption" sx={{ fontWeight: 600, color: theme.palette.text.secondary }}>
                        {policy.messages.length} finding{policy.messages.length === 1 ? '' : 's'}
                    </Typography>
                    {policy.messages.map((message, i) => (
                        <Box key={i} sx={{ display: 'flex', gap: 1, alignItems: 'flex-start' }}>
                            <Box
                                component="span"
                                sx={{
                                    width: 4,
                                    height: 4,
                                    mt: '7px',
                                    flexShrink: 0,
                                    borderRadius: '9999px',
                                    background: theme.palette.text.disabled,
                                }}
                            />
                            <Typography variant="body2" sx={{ color: theme.palette.text.secondary, whiteSpace: 'pre-wrap' }}>
                                {message}
                            </Typography>
                        </Box>
                    ))}
                </Box>
            )}

            <Box
                sx={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    gap: 1.5,
                    flexWrap: 'wrap',
                    mt: 2
                }}
            >
                {/* Where the policy comes from. A policy is inherited by every namespace below the
                    group that defines it, so the group is what says which one that is. With the
                    policy deleted the snapshot no longer names its group, and the TRN stands in as
                    the only remaining way to identify what was evaluated. */}
                {policy.policy ? (
                    <Tooltip title="Namespace this policy is defined in">
                        <Box sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.75, minWidth: 0, color: theme.palette.text.disabled }}>
                            <FolderOutlinedIcon sx={{ width: 14, height: 14, flexShrink: 0 }} />
                            <Typography variant="body2" sx={{ color: theme.palette.text.secondary, wordBreak: 'break-all' }}>
                                {policy.policy.groupPath}
                            </Typography>
                        </Box>
                    </Tooltip>
                ) : (
                    <Tooltip title="This policy is no longer available">
                        <Box sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.75, minWidth: 0, color: theme.palette.text.disabled }}>
                            <Typography variant="code" sx={{ color: theme.palette.text.secondary, wordBreak: 'break-all' }}>
                                {policy.provenance.policyTrn}
                            </Typography>
                            {/* Says outright why there is a TRN here instead of a namespace and a
                                link, rather than leaving it to the tooltip. */}
                            <Typography variant="body2" sx={{ flexShrink: 0 }}>(deleted)</Typography>
                        </Box>
                    </Tooltip>
                )}
                {policy.policy && (
                    <Typography
                        variant="body2"
                        component={RouterLink}
                        to={`/groups/${policy.policy.groupPath}/-/policies/${policy.policy.id}`}
                        sx={{
                            display: 'inline-flex',
                            alignItems: 'center',
                            gap: '5px',
                            fontWeight: 600,
                            color: theme.palette.primary.main,
                            textDecoration: 'none',
                            flexShrink: 0,
                            '&:hover': { textDecoration: 'underline' },
                        }}
                    >
                        View policy details
                        <ChevronRightIcon sx={{ width: 14, height: 14 }} />
                    </Typography>
                )}
            </Box>
        </Paper>
    );
}

export default RunTaskStagePolicyCheckPolicyCard;
