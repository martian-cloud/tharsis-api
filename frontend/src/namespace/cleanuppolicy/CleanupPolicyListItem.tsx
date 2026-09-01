import { Box, Button, Chip, Collapse, IconButton, Link, Paper, Stack, Tooltip, Typography } from '@mui/material';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { Link as RouterLink } from 'react-router-dom';
import Timestamp from '../../common/Timestamp';
import TRNButton from '../../common/TRNButton';
import { CLEANUP_POLICY_KINDS } from './rules';
import { CleanupPolicyKind } from './types';
import { policyRules } from './utils';
import { CleanupPolicyListItem_fields$key } from './__generated__/CleanupPolicyListItem_fields.graphql';

// Also doubles as the data contract mutations spread to normalize the Relay store.
const policyFragment = graphql`
    fragment CleanupPolicyListItem_fields on CleanupPolicy {
        id
        kind
        namespacePath
        disabled
        metadata { trn }
        lastSweepCompletedAt
        terraformModulePolicyData {
            rules {
                strategy
                description
                nameGlob
                systemGlob
                versionGlob
                deleteAfterDays
            }
        }
        terraformProviderPolicyData {
            rules {
                strategy
                description
                nameGlob
                versionGlob
                deleteAfterDays
            }
        }
        runPolicyData {
            rules {
                strategy
                description
                speculative
                assessment
                status
                keepMin
                deleteAfterDays
            }
        }
    }
`;

interface Props {
    policyRef: CleanupPolicyListItem_fields$key;
    // The namespace currently being viewed — compared against the policy's own namespacePath to
    // tell whether it's local here or inherited from a parent.
    namespacePath: string;
    label: string;
    description: string;
    expanded: boolean;
    onToggle: () => void;
}

// One row in the cleanup policy list — a collapsible panel summarizing a single kind's active
// policy (local or inherited), with its rules shown on expand.
function CleanupPolicyListItem({
    policyRef,
    namespacePath,
    label,
    description,
    expanded,
    onToggle,
}: Props) {
    const policy = useFragment(policyFragment, policyRef);
    const kind = policy.kind as CleanupPolicyKind;
    const kindDef = CLEANUP_POLICY_KINDS[kind];
    const rules = policyRules(policy, kindDef.policyDataKey);
    const inheritedFrom = policy.namespacePath !== namespacePath ? policy.namespacePath : undefined;
    const disabled = policy.disabled;
    const trn = policy.metadata?.trn ?? undefined;
    const lastSweepCompletedAt = policy.lastSweepCompletedAt ?? undefined;

    return (
        <Box sx={{ border: 1, borderRadius: 1, borderColor: 'divider' }}>
            <Box
                sx={{
                    display: 'flex', flexDirection: { xs: 'column', sm: 'row' },
                    alignItems: { xs: 'stretch', sm: 'center' },
                    gap: 1.5, p: 2, cursor: 'pointer',
                }}
                onClick={onToggle}
            >
                <Box sx={{ minWidth: 0, flexGrow: { sm: 1 } }}>
                    <Stack direction="row" alignItems="center" spacing={1} flexWrap="wrap">
                        <Typography variant="subtitle1">{label}</Typography>
                        {disabled && (
                            <Chip size="small" color="warning" label="Disabled" />
                        )}
                        {rules.length > 0 && (
                            <Chip
                                size="small" variant="outlined"
                                label={lastSweepCompletedAt
                                    ? <>Last swept <Timestamp timestamp={lastSweepCompletedAt} format="relative" /></>
                                    : 'Last swept never'}
                            />
                        )}
                    </Stack>
                    <Typography variant="body2" color="textSecondary">{description}</Typography>
                </Box>
                {/* stopPropagation prevents buttons and TRN from toggling the accordion */}
                <Box
                    sx={{
                        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                        gap: 1.5, flexShrink: 0,
                        width: { xs: '100%', sm: 'auto' },
                    }}
                    onClick={e => e.stopPropagation()}
                >
                    <Stack
                        direction={{ xs: 'column', sm: 'row' }}
                        spacing={1}
                        alignItems={{ xs: 'stretch', sm: 'center' }}
                        sx={{ flexGrow: { xs: 1, sm: 0 } }}
                    >
                        {trn && <TRNButton trn={trn} size="small" sx={{ width: { xs: '100%', sm: 'auto' } }} />}
                        {inheritedFrom ? (
                            <Tooltip title="Create a policy that takes precedence over the inherited one">
                                <Button
                                    size="small" variant="outlined"
                                    component={RouterLink}
                                    to={`new?kind=${kind.toLowerCase()}`}
                                >
                                    Override Policy
                                </Button>
                            </Tooltip>
                        ) : (
                            <Tooltip title="Edit the cleanup policy">
                                <Button
                                    size="small" variant="outlined"
                                    component={RouterLink}
                                    to={`${kind.toLowerCase()}/edit`}
                                >
                                    Edit
                                </Button>
                            </Tooltip>
                        )}
                    </Stack>
                    <Tooltip title={expanded ? 'Collapse' : 'Expand'}>
                        <IconButton size="small" onClick={onToggle} sx={{ flexShrink: 0 }}>
                            {expanded ? <ExpandLessIcon /> : <ExpandMoreIcon />}
                        </IconButton>
                    </Tooltip>
                </Box>
            </Box>

            <Collapse in={expanded}>
                <Box sx={{ px: 2, pb: 2 }}>
                    {rules.length === 0 ? (
                        <Paper sx={{ p: 2 }}>
                            <Typography>No cleanup rules are configured.</Typography>
                        </Paper>
                    ) : (
                        // onChange is omitted — CleanupRuleList treats absent onChange as read-only.
                        kindDef.renderList({
                            rules, onChange: undefined,
                            adding: undefined,
                            onAddingChange: () => { },
                            onRequestAdd: () => { },
                        })
                    )}
                    {inheritedFrom && (
                        <Box sx={{ mt: 1.5 }}>
                            <Stack direction="row" alignItems="center" spacing={0.5} sx={{ mt: 1.5, px: 0.5 }}>
                                <Typography variant="caption" color="textSecondary">
                                    Inherited from
                                </Typography>
                                <Link component={RouterLink} to={`/groups/${inheritedFrom}/-/cleanup_policies?expand=${kind.toLowerCase()}`} variant="caption" underline="hover">
                                    {inheritedFrom}
                                </Link>
                            </Stack>
                        </Box>
                    )}
                </Box>
            </Collapse >
        </Box >
    );
}

export default CleanupPolicyListItem;
