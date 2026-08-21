import MoreVertIcon from '@mui/icons-material/MoreVert';
import { Box, IconButton, Menu, MenuItem, Paper, Tooltip, Typography, useTheme } from '@mui/material';
import { type Theme } from '@mui/material/styles';
import graphql from 'babel-plugin-relay/macro';
import { useState } from 'react';
import { useFragment } from 'react-relay/hooks';
import { Link as RouterLink, useNavigate } from 'react-router-dom';
import Pill from '../../common/Pill';
import Link from '../../routes/Link';
import { PolicyCardFragment_policy$key } from './__generated__/PolicyCardFragment_policy.graphql';
import { ENFORCEMENT_LEVEL_LABELS, STAGE_LABELS } from './PolicyForm';
import { type ScopeRuleActionValue, scopeActionColor } from './scopeRules';

const FIELD_LABEL_SX = {
    fontWeight: 600,
    textTransform: 'uppercase' as const,
    letterSpacing: '0.07em',
    mb: '4px',
};

function enforcementLevelColor(level: string, theme: Theme): string {
    const l = level.toLowerCase();
    if (l === 'hard_mandatory') {
        return theme.palette.error.main;
    }
    if (l === 'soft_mandatory') {
        return theme.palette.warning.main;
    }
    return theme.palette.info.main;
}

// How many scope rules of one action a policy has, coloured to match the action's pill on the policy
// detail page. Only rendered for a non-zero count: a policy with no exclusions says nothing about
// them. A policy with a scope always has at least one rule, so at least one count is always shown.
function ScopeCount({ action, count }: { action: ScopeRuleActionValue, count: number }) {
    const theme = useTheme();
    return (
        <Box component="span" sx={{ display: 'inline-flex', alignItems: 'baseline', gap: '5px', fontSize: theme.typography.body2.fontSize }}>
            <Box component="span" sx={{ fontWeight: 700, color: scopeActionColor(action, theme) }}>
                {count}
            </Box>
            <Box component="span" sx={{ color: theme.palette.text.secondary }}>
                {action.toLowerCase()}
            </Box>
        </Box>
    );
}

interface Props {
    fragmentRef: PolicyCardFragment_policy$key;
    showGroupPath: boolean;
    showActions?: boolean;
    onDelete?: (policy: { id: string, name: string }) => void;
}

function PolicyCard({ fragmentRef, showGroupPath, showActions = true, onDelete }: Props) {
    const theme = useTheme();
    const navigate = useNavigate();
    const [menuAnchor, setMenuAnchor] = useState<HTMLElement | null>(null);

    const policy = useFragment<PolicyCardFragment_policy$key>(graphql`
        fragment PolicyCardFragment_policy on Policy {
            id
            name
            description
            kind
            createdBy
            requiredApprovals
            opaData {
                packageSource
                packageVersionConstraint
                stage
                enforcementLevel
            }
            scope {
                action
            }
            groupPath
            allowedUsers { id }
            allowedTeams { id }
            allowedServiceAccounts { id }
        }
    `, fragmentRef);

    const ownerPath = policy.groupPath;
    const opa = policy.opaData;
    const enfColor = opa ? enforcementLevelColor(opa.enforcementLevel, theme) : undefined;
    const enfLabel = opa ? (ENFORCEMENT_LEVEL_LABELS[opa.enforcementLevel] ?? opa.enforcementLevel) : '';

    // Approval only ever clears a soft-mandatory failure: nothing else can be overridden that way, so
    // no other enforcement level requires approvers at all. Named for what it means rather than for
    // what it used to gate — the section itself is always rendered now.
    const usesApprovals = opa?.enforcementLevel === 'SOFT_MANDATORY';
    const allowedUsers = policy.allowedUsers ?? [];
    const allowedTeams = policy.allowedTeams ?? [];
    const allowedServiceAccounts = policy.allowedServiceAccounts ?? [];
    const approverCount = allowedUsers.length + allowedTeams.length + allowedServiceAccounts.length;

    // A soft-mandatory policy with nobody listed also requires no approvals, but for a reason worth
    // stating: an override is then the only way past a failure.
    const requiresApprovals = usesApprovals && approverCount > 0;
    const approvalsSummary = requiresApprovals
        ? `Requires ${policy.requiredApprovals} out of ${approverCount} approver${approverCount === 1 ? '' : 's'}`
        : usesApprovals
            ? 'None — a failure can only be cleared by an override.'
            : 'None';

    const scope = policy.scope ?? [];
    const excludeCount = scope.filter(rule => rule.action === 'EXCLUDE').length;
    // Anything that is not an exclusion is an inclusion, so counting the remainder keeps the two
    // numbers adding up to the rule count even if another action is ever added to the enum.
    const includeCount = scope.length - excludeCount;

    return (
        <Paper
            variant="outlined"
            component="article"
            sx={{
                padding: 2,
                display: 'flex',
                flexDirection: 'column',
                // Lets the card shrink to its grid track instead of being pushed wider by unbreakable
                // content, which is what makes the ellipsis on the scope target below take effect.
                minWidth: 0,
                '&:hover': {
                    borderColor: theme.palette.action.hover,
                },
            }}
        >
            {/* Header */}
            <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 1 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap', minWidth: 0 }}>
                    <Typography
                        component={RouterLink}
                        to={`/groups/${ownerPath}/-/policies/${policy.id}`}
                        sx={{
                            fontWeight: 700,
                            color: theme.palette.text.primary,
                            textDecoration: 'none',
                            '&:hover': { textDecoration: 'underline' },
                        }}
                    >
                        {policy.name}
                    </Typography>
                    {opa && (
                        <Pill variant="tint" size="small" color={enfColor}>
                            {enfLabel}
                        </Pill>
                    )}
                </Box>
                {showActions && (
                    <>
                        <IconButton
                            size="small"
                            sx={{ flexShrink: 0, borderRadius: '7px' }}
                            onClick={e => { e.preventDefault(); setMenuAnchor(e.currentTarget); }}
                        >
                            <MoreVertIcon fontSize="small" />
                        </IconButton>
                        <Menu
                            anchorEl={menuAnchor}
                            open={Boolean(menuAnchor)}
                            onClose={() => setMenuAnchor(null)}
                            anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
                            transformOrigin={{ vertical: 'top', horizontal: 'right' }}
                        >
                            <MenuItem onClick={() => { setMenuAnchor(null); navigate(`/groups/${ownerPath}/-/policies/${policy.id}/edit`); }}>
                                Edit
                            </MenuItem>
                            {onDelete && (
                                <MenuItem
                                    onClick={() => { setMenuAnchor(null); onDelete({ id: policy.id, name: policy.name }); }}
                                    sx={{ color: theme.palette.error.main }}
                                >
                                    Delete
                                </MenuItem>
                            )}
                        </Menu>
                    </>
                )}
            </Box>

            {/* Description */}
            {policy.description && (
                <Typography
                    variant="body2"
                    color="textSecondary"
                    sx={{ mt: 1.5, lineHeight: 1.5 }}
                >
                    {policy.description}
                </Typography>
            )}

            {/* Field grid */}
            <Box
                sx={{
                    mt: 2.5,
                    display: 'grid',
                    gridTemplateColumns: '1fr 1fr',
                    gap: '16px 18px',
                }}
            >
                <Box>
                    <Typography variant="caption" component="div" sx={{ ...FIELD_LABEL_SX, color: theme.palette.text.secondary }}>
                        Policy Type
                    </Typography>
                    <Typography variant="code" sx={{ color: theme.palette.text.primary }}>
                        {policy.kind}
                    </Typography>
                </Box>
                <Box>
                    <Typography variant="caption" component="div" sx={{ ...FIELD_LABEL_SX, color: theme.palette.text.secondary }}>
                        Run Stage
                    </Typography>
                    <Typography variant="code" sx={{ color: theme.palette.text.primary }}>
                        {opa?.stage ? (STAGE_LABELS[opa.stage] ?? opa.stage) : '—'}
                    </Typography>
                </Box>
                <Box sx={{ gridColumn: '1 / 3'}}>
                    <Typography variant="caption" component="div" sx={{ ...FIELD_LABEL_SX, color: theme.palette.text.secondary }}>
                        Package Source
                    </Typography>
                    <Typography variant="code" sx={{ color: theme.palette.text.primary }}>
                        {opa?.packageSource ?? '—'}
                    </Typography>
                </Box>
                <Box sx={{ gridColumn: '1 / 3'}}>
                    <Typography variant="caption" component="div" sx={{ ...FIELD_LABEL_SX, color: theme.palette.text.secondary }}>
                        Package Version
                    </Typography>
                    <Typography variant="code" sx={{ color: theme.palette.text.primary }}>
                        {opa?.packageVersionConstraint ?? 'latest'}
                    </Typography>
                </Box>
            </Box>

            {/* Scope: which run contexts the policy applies to. Unconditional, unlike the approvers
                below — every policy has a scope, and an empty one is meaningful rather than absent. */}
            <Box
                sx={{
                    mt: 2.5,
                    pt: '12px',
                    borderTop: `1px solid ${theme.palette.divider}`,
                    minWidth: 0,
                }}
            >
                <Typography variant="caption" component="div" sx={{ ...FIELD_LABEL_SX, color: theme.palette.text.secondary }}>
                    Scope Rules
                </Typography>
                {scope.length === 0 ? (
                    // Deliberately not "0 rules": an empty scope is the widest scope there is, and a
                    // bare zero would read as the exact opposite. Worded as PolicyDetails words the
                    // same case.
                    <Typography variant="caption" color="textSecondary">
                        Applies to all workspaces under the owning group.
                    </Typography>
                ) : (
                    <Box sx={{ display: 'flex', alignItems: 'baseline', gap: 2, flexWrap: 'wrap' }}>
                        {includeCount > 0 && <ScopeCount action="INCLUDE" count={includeCount} />}
                        {excludeCount > 0 && <ScopeCount action="EXCLUDE" count={excludeCount} />}
                    </Box>
                )}
            </Box>

            {/* Approvers: who may override a soft-mandatory failure. Rendered for every policy, so that
                requiring no approvals is stated outright rather than left to be inferred from a section
                that isn't there. Who they are is the detail page's job; the card only answers how many
                of how many, which is what decides whether a failure can realistically be cleared. */}
            <Box
                sx={{
                    mt: 2.5,
                    pt: '12px',
                    borderTop: `1px solid ${theme.palette.divider}`,
                    minWidth: 0,
                }}
            >
                <Typography variant="caption" component="div" sx={{ ...FIELD_LABEL_SX, color: theme.palette.text.secondary }}>
                    Required Approvers
                </Typography>
                <Typography
                    variant="body2"
                    sx={{ color: requiresApprovals ? theme.palette.text.primary : theme.palette.text.secondary }}
                >
                    {approvalsSummary}
                </Typography>
            </Box>

            {showGroupPath && (
                <>
                    <Box sx={{ flex: 1 }} />
                    <Box
                        sx={{
                            mt: 2.5,
                            pt: '12px',
                            minWidth: 0,
                            borderTop: `1px solid ${theme.palette.divider}`,
                        }}
                    >
                        <Typography variant="caption" component="div" sx={{ ...FIELD_LABEL_SX, color: theme.palette.text.secondary }}>
                            Group
                        </Typography>
                        <Tooltip title={ownerPath}>
                            <Link
                                to={`/groups/${ownerPath}`}
                                variant="body2"
                                color="textSecondary"
                                noWrap
                                // Block rather than the anchor's default inline: noWrap truncates with
                                // text-overflow, which needs a block box to clip against, and the path
                                // being too long for its track is the normal case here.
                                sx={{ display: 'block' }}
                            >
                                {ownerPath}
                            </Link>
                        </Tooltip>
                    </Box>
                </>
            )}
        </Paper>
    );
}

export default PolicyCard;
