import ArrowDropDownIcon from '@mui/icons-material/ArrowDropDown';
import { Avatar, Box, ButtonGroup, Menu, MenuItem, Paper, Typography, useTheme } from '@mui/material';
import Button from '@mui/material/Button';
import { alpha, type Theme } from '@mui/material/styles';
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import { ReactNode, useState } from 'react';
import { useFragment, useLazyLoadQuery, useMutation } from 'react-relay/hooks';
import { useNavigate, useParams } from 'react-router-dom';
import { ConnectionHandler } from 'relay-runtime';
import ConfirmationDialog from '../../common/ConfirmationDialog';
import Gravatar from '../../common/Gravatar';
import Pill from '../../common/Pill';
import NamespaceBreadcrumbs from '../NamespaceBreadcrumbs';
import { ENFORCEMENT_LEVEL_LABELS, KIND_LABELS, STAGE_LABELS } from './policyDisplay';
import { SCOPE_TYPE_LABELS, scopeActionColor } from './scopeRules';
import { PolicyDetailsDeleteMutation } from './__generated__/PolicyDetailsDeleteMutation.graphql';
import { PolicyDetailsFragment_policy$key } from './__generated__/PolicyDetailsFragment_policy.graphql';
import { PolicyDetailsQuery } from './__generated__/PolicyDetailsQuery.graphql';

// The policy is loaded by id rather than handed down from the list: this view is reachable by url, and
// the fields only it renders are its own business rather than something every row in the list has to
// carry along on the chance one of them gets opened.
const query = graphql`
    query PolicyDetailsQuery($id: String!) {
        node(id: $id) {
            ... on Policy {
                ...PolicyDetailsFragment_policy
            }
        }
    }
`;

function scopePillSx(action: string, theme: Theme) {
    const color = scopeActionColor(action, theme);
    return { background: alpha(color, 0.18), color };
}

const SECTION_LABEL_SX = {
    fontWeight: 700,
    textTransform: 'uppercase',
    letterSpacing: '0.09em',
} as const;

const FIELD_LABEL_SX = {
    fontWeight: 600,
    textTransform: 'uppercase',
    letterSpacing: '0.08em',
    mb: '7px',
} as const;

// SectionHeader separates the card's field groups. Every section but the first is preceded by a rule,
// which is what gives the card its stacked-panel look without nesting more borders.
function SectionHeader({ children, divider = true, mb = '18px' }: { children: ReactNode, divider?: boolean, mb?: string }) {
    const theme = useTheme();
    return (
        <Box
            sx={{
                ...(divider ? { mt: '30px', pt: '26px', borderTop: `1px solid ${theme.palette.divider}` } : {}),
                mb,
            }}
        >
            <Typography variant="caption" component="div" sx={{ ...SECTION_LABEL_SX, color: theme.palette.text.secondary }}>
                {children}
            </Typography>
        </Box>
    );
}

function Field({ label, value, mono }: { label: string, value: ReactNode, mono?: boolean }) {
    const theme = useTheme();
    return (
        <Box>
            <Typography variant="caption" component="div" sx={{ ...FIELD_LABEL_SX, color: theme.palette.text.secondary }}>
                {label}
            </Typography>
            <Typography
                variant={mono ? 'code' : 'body1'}
                component="div"
                sx={{
                    color: theme.palette.text.primary,
                    wordBreak: 'break-word',
                }}
            >
                {value}
            </Typography>
        </Box>
    );
}

function FieldGrid({ children }: { children: ReactNode }) {
    const theme = useTheme();
    return (
        <Box
            sx={{
                display: 'grid',
                gridTemplateColumns: 'repeat(3, 1fr)',
                gap: '28px 40px',
                [theme.breakpoints.down('md')]: { gridTemplateColumns: '1fr' },
            }}
        >
            {children}
        </Box>
    );
}

// An approver row: users get their gravatar, teams and service accounts a monogram, matching how the
// policy form renders the same principals.
function ApproverRow({ email, initial, name }: { email?: string, initial?: string, name: string }) {
    const theme = useTheme();
    return (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
            {email
                ? <Gravatar email={email} width={34} height={34} />
                : <Avatar sx={{ width: 34, height: 34, fontSize: theme.typography.caption.fontSize, fontWeight: 700 }}>{initial}</Avatar>}
            <Typography variant="subtitle2" component="div" sx={{ color: theme.palette.text.primary }} noWrap>
                {name}
            </Typography>
        </Box>
    );
}

interface Props {
    // The group the policy list connection is rooted at, so the connection id can be reconstructed
    // deterministically without the caller having to load and hand over its own copy of the list query.
    ownerId: string;
    // The namespace being viewed, for the breadcrumbs.
    ownerPath: string;
    // The group a policy must be owned by to count as local rather than inherited. Deliberately not
    // ownerPath: under a workspace that is the workspace's own path, which no policy's group can ever
    // equal, so every policy would read as inherited.
    currentGroupPath: string;
}

function PolicyDetails({ ownerId, ownerPath, currentGroupPath }: Props) {
    const { policyId } = useParams<{ policyId: string }>();
    const navigate = useNavigate();
    const theme = useTheme();
    const { enqueueSnackbar } = useSnackbar();
    const [showDeleteDialog, setShowDeleteDialog] = useState(false);
    const [menuAnchorEl, setMenuAnchorEl] = useState<Element | null>(null);

    // Reconstructed rather than handed down from PolicyList: the two views are independent routes, and
    // this way neither needs to run the other's query to know the connection's store id.
    const connectionId = ConnectionHandler.getConnectionID(
        ownerId,
        'PolicyList_policies',
        { includeInherited: true, sort: 'CREATED_AT_DESC' }
    );

    const queryData = useLazyLoadQuery<PolicyDetailsQuery>(query, { id: policyId ?? '' }, { fetchPolicy: 'store-and-network' });

    const policy = useFragment<PolicyDetailsFragment_policy$key>(graphql`
        fragment PolicyDetailsFragment_policy on Policy {
            id
            name
            description
            kind
            createdBy
            requiredApprovals
            opaData {
                packageSource
                packageVersionConstraint
                packageDigest
                stage
                enforcementLevel
                speculativeRunEnforcementLevel
            }
            moduleAttestationData {
                publicKey
                predicateType
                verifyStateLineage
                stage
                enforcementLevel
                speculativeRunEnforcementLevel
            }
            scope {
                type
                action
                pattern
            }
            # The owning group. For an inherited policy that is an ancestor of the namespace being
            # viewed rather than the namespace itself. Read off the policy's own TRN, so naming the
            # owner costs no group lookup.
            groupPath
            allowedUsers { id email username }
            allowedTeams { id name }
            allowedServiceAccounts { id name resourcePath }
        }
    `, queryData.node);

    const [commitDelete, commitDeleteInFlight] = useMutation<PolicyDetailsDeleteMutation>(graphql`
        mutation PolicyDetailsDeleteMutation($input: DeletePolicyInput!, $connections: [ID!]!) {
            deletePolicy(input: $input) {
                policy {
                    id @deleteEdge(connections: $connections)
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    if (!policy) {
        return null;
    }

    const ownerGroupPath = policy.groupPath;
    const inherited = ownerGroupPath !== currentGroupPath;
    const opa = policy.opaData;
    const attestation = policy.moduleAttestationData;
    // The two kind-specific data objects are mutually exclusive, so whichever is present carries the
    // fields (stage, enforcement levels) that are common in shape but stored per kind.
    const kindData = opa ?? attestation;
    const enforcementLevel = kindData?.enforcementLevel ?? '';
    const enforcementLevelColors = theme.palette.enforcementLevel;
    const enforcementLevelColor = enforcementLevelColors[enforcementLevel as keyof typeof enforcementLevelColors] ?? enforcementLevelColors.ADVISORY;
    const speculativeLevel = kindData?.speculativeRunEnforcementLevel ?? '';
    // requiredApprovals and the allowed-subject lists only mean anything for a soft-mandatory policy:
    // advisory failures never block and hard-mandatory ones can never be overridden.
    const showApprovals = enforcementLevel === 'SOFT_MANDATORY';
    const allowedUsers = policy.allowedUsers ?? [];
    const allowedTeams = policy.allowedTeams ?? [];
    const allowedServiceAccounts = policy.allowedServiceAccounts ?? [];
    const scope = policy.scope ?? [];
    const approverCount = allowedUsers.length + allowedTeams.length + allowedServiceAccounts.length;

    const onDeleteConfirm = () => {
        commitDelete({
            variables: {
                // @deleteEdge drops the policy's card from the list this view navigates back to.
                input: { id: policy.id },
                connections: [connectionId],
            },
            onCompleted: data => {
                setShowDeleteDialog(false);
                if (data.deletePolicy.problems.length) {
                    enqueueSnackbar(data.deletePolicy.problems.map(p => p.message).join('; '), { variant: 'warning' });
                } else {
                    navigate('..');
                }
            },
            onError: err => {
                setShowDeleteDialog(false);
                enqueueSnackbar(`Unexpected error occurred: ${err.message}`, { variant: 'error' });
            },
        });
    };

    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={ownerPath}
                childRoutes={[
                    { title: 'policies', path: 'policies' },
                    { title: policy.name, path: policy.id },
                ]}
            />

            {/* Header */}
            <Box sx={{
                display: 'flex',
                alignItems: 'flex-start',
                justifyContent: 'space-between',
                flexDirection: { xs: 'column', sm: 'row' },
                gap: 3,
                mb: '22px',
            }}>
                <Box sx={{ minWidth: 0 }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, flexWrap: 'wrap' }}>
                        <Typography
                            variant="h5"
                            component="h1"
                            sx={{ m: 0, fontWeight: 700, color: theme.palette.text.primary }}
                        >
                            {policy.name}
                        </Typography>
                        {enforcementLevel && (
                            <Pill variant="tint" size="small" color={enforcementLevelColor}>
                                {ENFORCEMENT_LEVEL_LABELS[enforcementLevel] ?? enforcementLevel}
                            </Pill>
                        )}
                    </Box>
                    {policy.description && (
                        <Typography
                            sx={{
                                mt: '14px',
                                lineHeight: 1.6,
                                color: theme.palette.text.secondary,
                                maxWidth: 680,
                            }}
                        >
                            {policy.description}
                        </Typography>
                    )}
                    {inherited && (
                        <Typography color="textSecondary" variant="body2" sx={{ mt: 1 }}>
                            Inherited from <strong>{ownerGroupPath}</strong>
                        </Typography>
                    )}
                </Box>
                {!inherited && (
                    <Box sx={{ display: 'flex', alignItems: 'center', flexShrink: 0 }}>
                        {/* Edit and the overflow menu read as one control, matching the details pages for
                            service accounts, managed identities and runners. */}
                        <ButtonGroup variant="outlined" color="primary" sx={{ height: 38 }}>
                            <Button
                                onClick={() => navigate('edit')}
                                sx={{ px: '18px', fontWeight: 600, letterSpacing: '0.05em' }}
                            >
                                Edit
                            </Button>
                            <Button
                                size="small"
                                aria-label="more options menu"
                                aria-haspopup="menu"
                                onClick={event => setMenuAnchorEl(event.currentTarget)}
                            >
                                <ArrowDropDownIcon fontSize="small" />
                            </Button>
                        </ButtonGroup>
                        <Menu
                            anchorEl={menuAnchorEl}
                            open={Boolean(menuAnchorEl)}
                            onClose={() => setMenuAnchorEl(null)}
                            anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
                            transformOrigin={{ vertical: 'top', horizontal: 'right' }}
                        >
                            <MenuItem
                                onClick={() => {
                                    setMenuAnchorEl(null);
                                    setShowDeleteDialog(true);
                                }}
                                sx={{ color: 'error.main' }}
                            >
                                Delete Policy
                            </MenuItem>
                        </Menu>
                    </Box>
                )}
            </Box>

            <Paper variant="outlined" sx={{ padding: 3 }}>
                <SectionHeader divider={false}>General</SectionHeader>
                <FieldGrid>
                    <Field label="Policy Type" value={KIND_LABELS[policy.kind] ?? policy.kind} />
                    <Field label="Run Stage" value={kindData?.stage ? (STAGE_LABELS[kindData.stage] ?? kindData.stage) : '—'} mono />
                    <Field label="Created By" value={policy.createdBy} />
                </FieldGrid>

                <SectionHeader>Enforcement Level</SectionHeader>
                <FieldGrid>
                    <Field label="Apply Runs" value={enforcementLevel ? (ENFORCEMENT_LEVEL_LABELS[enforcementLevel] ?? enforcementLevel) : '—'} />
                    <Field label="Speculative & Assessment Runs" value={speculativeLevel ? (ENFORCEMENT_LEVEL_LABELS[speculativeLevel] ?? speculativeLevel) : '—'} />
                </FieldGrid>

                {opa && <>
                    <SectionHeader>Package</SectionHeader>
                    <FieldGrid>
                        <Field label="Package Source" value={opa.packageSource} mono />
                        <Field label="Package Version" value={opa.packageVersionConstraint || 'latest'} mono />
                        {/* The digest spans the grid — it is too long to sit in a third of the row — and
                            carries the LOCKED marker on its label, because a pinned digest is what makes
                            the version above immovable. It is always shown: whether the package floats or
                            is pinned is the point, so an absent digest has to say so rather than vanish. */}
                        <Box sx={{ gridColumn: '1 / -1' }}>
                            <Box sx={{ display: 'flex', alignItems: 'center', gap: '9px', mb: '7px' }}>
                                <Typography variant="caption" component="div" sx={{ ...FIELD_LABEL_SX, mb: 0, color: theme.palette.text.secondary }}>
                                    Package Digest
                                </Typography>
                                <Box
                                    component="span"
                                    sx={{
                                        display: 'inline-flex',
                                        alignItems: 'center',
                                        padding: '2px 9px',
                                        borderRadius: '5px',
                                        fontSize: theme.typography.caption.fontSize,
                                        fontWeight: 700,
                                        letterSpacing: '0.05em',
                                        ...(opa.packageDigest
                                            ? { background: alpha(theme.palette.primary.main, 0.14), color: theme.palette.primary.main }
                                            : { background: alpha(theme.palette.text.secondary, 0.14), color: theme.palette.text.secondary }),
                                    }}
                                >
                                    {opa.packageDigest ? 'LOCKED' : 'UNPINNED'}
                                </Box>
                            </Box>
                            {opa.packageDigest ? (
                                <Typography variant="code" component="div" sx={{ color: theme.palette.text.primary, wordBreak: 'break-all', lineHeight: 1.5 }}>
                                    {opa.packageDigest}
                                </Typography>
                            ) : (
                                <Typography variant="body2" sx={{ color: theme.palette.text.secondary, lineHeight: 1.5 }}>
                                    Not set — the package is resolved by version at run creation.
                                </Typography>
                            )}
                        </Box>
                    </FieldGrid>
                </>}

                {attestation && <>
                    <SectionHeader>Module Attestation</SectionHeader>
                    <FieldGrid>
                        <Field label="Predicate Type" value={attestation.predicateType || 'Any'} mono />
                        <Field label="Verify State Lineage" value={attestation.verifyStateLineage ? 'Yes' : 'No'} />
                        <Box sx={{ gridColumn: '1 / -1' }}>
                            <Typography variant="caption" component="div" sx={{ ...FIELD_LABEL_SX, color: theme.palette.text.secondary }}>
                                Public Key
                            </Typography>
                            <Typography
                                variant="code"
                                component="div"
                                sx={{ color: theme.palette.text.primary, wordBreak: 'break-all', whiteSpace: 'pre-wrap', lineHeight: 1.5 }}
                            >
                                {attestation.publicKey}
                            </Typography>
                        </Box>
                    </FieldGrid>
                </>}

                <SectionHeader mb="14px">Scope Rules</SectionHeader>
                {scope.length === 0 ? (
                    <Typography variant="body2" color="textSecondary">
                        No scope rules — applies to all workspaces under the owning group.
                    </Typography>
                ) : (
                    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '11px' }}>
                        {scope.map((rule, i) => (
                            <Box key={i} sx={{ display: 'flex', alignItems: 'center', gap: 1.5, flexWrap: 'wrap' }}>
                                <Box
                                    component="span"
                                    sx={{
                                        display: 'inline-flex',
                                        padding: '3px 11px',
                                        borderRadius: '6px',
                                        fontSize: theme.typography.caption.fontSize,
                                        fontWeight: 700,
                                        letterSpacing: '0.05em',
                                        flexShrink: 0,
                                        ...scopePillSx(rule.action, theme),
                                    }}
                                >
                                    {rule.action}
                                </Box>
                                <Typography variant="body2" sx={{ color: theme.palette.text.secondary }}>
                                    {SCOPE_TYPE_LABELS[rule.type] ?? rule.type}
                                </Typography>
                                <Typography variant="code" sx={{ color: theme.palette.text.primary }}>
                                    {rule.pattern}
                                </Typography>
                            </Box>
                        ))}
                    </Box>
                )}

                {showApprovals && (
                    <>
                        <Box sx={{
                            mt: '30px',
                            pt: '26px',
                            borderTop: `1px solid ${theme.palette.divider}`,
                            mb: '14px',
                            display: 'flex',
                            alignItems: 'baseline',
                            gap: 1.5,
                            flexWrap: 'wrap',
                        }}>
                            <Typography variant="caption" component="div" sx={{ ...SECTION_LABEL_SX, color: theme.palette.text.secondary }}>
                                Required Approvals
                            </Typography>
                            <Typography variant="caption" sx={{ color: theme.palette.text.secondary }}>
                                {policy.requiredApprovals} of {approverCount} approver{approverCount === 1 ? '' : 's'} required
                            </Typography>
                        </Box>
                        {approverCount === 0 ? (
                            <Typography variant="body2" color="textSecondary">
                                No approvers configured — a soft-mandatory failure can only be cleared by an override.
                            </Typography>
                        ) : (
                            <Box sx={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
                                {allowedUsers.map(user => (
                                    <ApproverRow key={user.id} email={user.email} name={user.username} />
                                ))}
                                {allowedTeams.map(team => (
                                    <ApproverRow key={team.id} initial={team.name.charAt(0).toUpperCase()} name={team.name} />
                                ))}
                                {allowedServiceAccounts.map(sa => (
                                    <ApproverRow key={sa.id} initial={sa.name.charAt(0).toUpperCase()} name={sa.resourcePath} />
                                ))}
                            </Box>
                        )}
                    </>
                )}
            </Paper>

            {showDeleteDialog && (
                <ConfirmationDialog
                    title="Remove Policy"
                    confirmLabel="Remove"
                    confirmInProgress={commitDeleteInFlight}
                    onConfirm={onDeleteConfirm}
                    onClose={() => setShowDeleteDialog(false)}
                >
                    Are you sure you want to remove the policy <strong>{policy.name}</strong>?
                </ConfirmationDialog>
            )}
        </Box>
    );
}

export default PolicyDetails;
