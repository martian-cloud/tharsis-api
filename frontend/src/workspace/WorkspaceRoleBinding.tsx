import { Alert, Box, Button, Divider, Paper, Typography, useTheme } from '@mui/material';
import { alpha } from '@mui/material/styles';
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import { KeyboardEvent, useEffect, useState } from 'react';
import { useFragment, useLazyLoadQuery, useMutation } from 'react-relay/hooks';
import { MutationError } from '../common/error';
import NamespaceBreadcrumbs from '../namespace/NamespaceBreadcrumbs';
import { RoleOption } from '../namespace/members/RoleAutocomplete';
import { WorkspaceRoleBindingFragment_workspace$key } from './__generated__/WorkspaceRoleBindingFragment_workspace.graphql';
import { WorkspaceRoleBindingQuery } from './__generated__/WorkspaceRoleBindingQuery.graphql';
import { WorkspaceRoleBindingRoleFragment_workspace$key } from './__generated__/WorkspaceRoleBindingRoleFragment_workspace.graphql';
import { WorkspaceRoleBindingSetMutation } from './__generated__/WorkspaceRoleBindingSetMutation.graphql';

interface Props {
    fragmentRef: WorkspaceRoleBindingFragment_workspace$key
}

function RoleDot({ selected }: { selected: boolean }) {
    const theme = useTheme();
    return (
        <Box sx={{
            width: 16,
            height: 16,
            borderRadius: '50%',
            flexShrink: 0,
            mt: '2px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            border: `1px solid ${selected ? theme.palette.primary.main : theme.palette.action.disabled}`,
        }}>
            <Box sx={{
                width: 8,
                height: 8,
                borderRadius: '50%',
                bgcolor: theme.palette.primary.main,
                transform: selected ? 'scale(1)' : 'scale(0)',
                transition: 'transform 160ms cubic-bezier(0.17,0.04,0.03,0.94)',
            }} />
        </Box>
    );
}

function CurrentBadge() {
    const theme = useTheme();
    return (
        <Box component="span" sx={{
            fontSize: 11,
            fontWeight: 600,
            letterSpacing: '0.05em',
            textTransform: 'uppercase',
            color: 'text.secondary',
            border: `1px solid ${theme.palette.divider}`,
            borderRadius: 999,
            px: 1,
            py: '1px',
        }}>
            current
        </Box>
    );
}

interface SelectableRowProps {
    selected: boolean
    onClick: () => void
    title: string
    mono?: boolean
    description: string
    badge?: boolean
}

function SelectableRow(props: SelectableRowProps) {
    const { selected, onClick, title, mono, description, badge } = props;
    const theme = useTheme();

    const onKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
        if (event.key === ' ' || event.key === 'Enter') {
            event.preventDefault();
            onClick();
        }
    };

    return (
        <Box
            role="radio"
            aria-checked={selected}
            tabIndex={0}
            onClick={onClick}
            onKeyDown={onKeyDown}
            sx={{
                display: 'flex',
                gap: 1.75,
                alignItems: 'flex-start',
                cursor: 'pointer',
                p: 1.75,
                borderRadius: 1.5,
                outline: 'none',
                border: `1px solid ${selected ? theme.palette.primary.main : theme.palette.divider}`,
                bgcolor: selected ? alpha(theme.palette.primary.main, 0.07) : 'transparent',
                transition: 'border-color 160ms, background 160ms',
                '&:hover': { borderColor: theme.palette.text.disabled, bgcolor: theme.palette.action.hover },
                '&:focus-visible': { borderColor: theme.palette.primary.main },
            }}
        >
            <RoleDot selected={selected} />
            <Box sx={{ flex: 1, minWidth: 0 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.25, flexWrap: 'wrap' }}>
                    <Typography sx={{ fontFamily: mono ? 'monospace' : 'inherit', fontSize: 14, fontWeight: mono ? 500 : 600 }}>
                        {title}
                    </Typography>
                    {badge && <CurrentBadge />}
                </Box>
                <Typography variant="body2" color="textSecondary" sx={{ mt: 0.5 }}>
                    {description}
                </Typography>
            </Box>
        </Box>
    );
}

function WorkspaceRoleBinding(props: Props) {
    const { enqueueSnackbar } = useSnackbar();
    const theme = useTheme();
    const [error, setError] = useState<MutationError | null>(null);

    const workspace = useFragment<WorkspaceRoleBindingFragment_workspace$key>(graphql`
        fragment WorkspaceRoleBindingFragment_workspace on Workspace {
            id
            name
            fullPath
            groupPath
        }
    `, props.fragmentRef);

    const [commit, commitInFlight] = useMutation<WorkspaceRoleBindingSetMutation>(graphql`
        mutation WorkspaceRoleBindingSetMutation($input: SetWorkspaceRoleBindingInput!) {
            setWorkspaceRoleBinding(input: $input) {
                workspace {
                    ...WorkspaceRoleBindingRoleFragment_workspace
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const queryData = useLazyLoadQuery<WorkspaceRoleBindingQuery>(graphql`
        query WorkspaceRoleBindingQuery($id: String!) {
            node(id: $id) {
                ...WorkspaceRoleBindingRoleFragment_workspace
            }
            # Only query for default roles
            roles(first: 5) {
                edges {
                    node {
                        id
                        name
                        description
                    }
                }
            }
        }
    `, { id: workspace.id }, { fetchPolicy: 'store-and-network' });

    const roleBindingData = useFragment<WorkspaceRoleBindingRoleFragment_workspace$key>(graphql`
        fragment WorkspaceRoleBindingRoleFragment_workspace on Workspace {
            roleBinding {
                id
                role {
                    id
                    name
                    description
                }
            }
        }
    `, queryData.node);

    const roles = queryData.roles.edges?.map(edge => edge?.node as RoleOption) ?? [];

    const currentRole: RoleOption | null = roleBindingData?.roleBinding ? {
        id: roleBindingData.roleBinding.role.id,
        name: roleBindingData.roleBinding.role.name,
        description: roleBindingData.roleBinding.role.description,
    } : null;
    const currentRoleId = currentRole?.id ?? null;

    const [pendingRole, setPendingRole] = useState<RoleOption | null>(currentRole);

    // Re-sync the pending selection whenever the committed binding changes (e.g. after a save).
    useEffect(() => {
        setPendingRole(currentRole);
    }, [currentRoleId]);

    const pendingId = pendingRole?.id ?? null;
    const dirty = pendingId !== currentRoleId;
    const removing = dirty && pendingId === null;

    const onCancel = () => {
        setError(null);
        setPendingRole(currentRole);
    };

    const onSave = () => {
        setError(null);
        commit({
            variables: {
                input: {
                    workspaceId: workspace.id,
                    roleId: pendingId,
                },
            },
            onCompleted: response => {
                if (response.setWorkspaceRoleBinding.problems.length) {
                    setError({
                        severity: 'warning',
                        message: response.setWorkspaceRoleBinding.problems.map(problem => problem.message).join('; ')
                    });
                } else {
                    enqueueSnackbar(removing ? 'Role binding removed' : 'Role binding updated', { variant: 'success' });
                }
            },
            onError: error => {
                setError({
                    severity: 'error',
                    message: `Unexpected error occurred: ${error.message}`
                });
            }
        });
    };

    const group = workspace.groupPath;

    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={workspace.fullPath}
                childRoutes={[
                    { title: "role binding", path: 'role_binding' }
                ]}
            />
            <Typography variant="h5" gutterBottom>Workspace Role Binding</Typography>
            <Typography variant="body2" color="textSecondary" sx={{ mb: 3 }}>
                A bound role lets runs in this workspace manage resources in group <strong>{group}</strong> through
                the Tharsis Terraform provider.
            </Typography>

            {!!pendingRole && (
                <Alert severity="info" sx={{ mb: 2.5 }}>
                    Anyone who can trigger a run in this workspace effectively holds <strong>{pendingRole.name}</strong> in group <strong>{group}</strong>.
                </Alert>
            )}

            {error && <Alert sx={{ mb: 2 }} severity={error.severity}>
                {error.message}
            </Alert>}

            <Paper variant="outlined">
                <Box sx={{
                    display: 'flex',
                    alignItems: 'baseline',
                    justifyContent: 'space-between',
                    gap: 2,
                    px: 2.5,
                    py: 2,
                    borderBottom: `1px solid ${theme.palette.divider}`,
                }}>
                    <Typography variant="caption" sx={{ fontWeight: 600, letterSpacing: '0.08em', textTransform: 'uppercase', color: 'text.secondary' }}>
                        Bound role
                    </Typography>
                    <Typography variant="caption" sx={{ fontFamily: 'monospace', color: 'text.disabled' }}>
                        {currentRole?.name ?? 'none'}
                    </Typography>
                </Box>

                <Box role="radiogroup" aria-label="Bound role">
                    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1, p: 2.5 }}>
                        {roles.map(role => (
                            <SelectableRow
                                key={role.id}
                                selected={pendingId === role.id}
                                mono
                                onClick={() => setPendingRole(role)}
                                title={role.name}
                                description={role.description || 'No description provided for this role.'}
                                badge={role.id === currentRoleId}
                            />
                        ))}
                    </Box>

                    <Divider />

                    <Box sx={{ p: 2.5 }}>
                        <SelectableRow
                            selected={pendingId === null}
                            onClick={() => setPendingRole(null)}
                            title="No role"
                            description={`Runs in this workspace can no longer manage Tharsis resources in ${group}.`}
                            badge={currentRoleId === null}
                        />
                    </Box>
                </Box>
            </Paper>

            <Divider light sx={{ mt: 4 }} />
            <Box sx={{ mt: 2, display: 'flex', alignItems: 'center', gap: 2, flexWrap: 'wrap' }}>
                <Button
                    variant="outlined"
                    color={"primary"}
                    disabled={!dirty}
                    loading={commitInFlight}
                    onClick={onSave}
                >
                    Save
                </Button>
                <Button color="inherit" disabled={!dirty} onClick={onCancel}>
                    Cancel
                </Button>
            </Box>
        </Box>
    );
}

export default WorkspaceRoleBinding;
