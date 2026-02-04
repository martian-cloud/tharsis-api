import ArrowDropDownIcon from '@mui/icons-material/ArrowDropDown';
import { ButtonGroup, Chip, Menu, MenuItem, Paper, Stack, styled, Table, TableBody, TableCell, TableHead, TableRow } from '@mui/material';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import Typography from '@mui/material/Typography';
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import React, { useState } from 'react';
import { useFragment, useLazyLoadQuery, useMutation } from "react-relay/hooks";
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { atomDark as prismTheme } from 'react-syntax-highlighter/dist/esm/styles/prism';
import config from '../../common/config';
import ConfirmationDialog from '../../common/ConfirmationDialog';
import TRNButton from '../../common/TRNButton';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import { ManagedIdentityDetailsDeleteAliasMutation } from './__generated__/ManagedIdentityDetailsDeleteAliasMutation.graphql';
import { ManagedIdentityDetailsDeleteMutation } from './__generated__/ManagedIdentityDetailsDeleteMutation.graphql';
import { ManagedIdentityDetailsFragment_group$key } from './__generated__/ManagedIdentityDetailsFragment_group.graphql';
import { ManagedIdentityDetailsQuery } from './__generated__/ManagedIdentityDetailsQuery.graphql';
import ManagedIdentityAliases from './aliases/ManagedIdentityAliases';
import { INITIAL_ITEM_COUNT } from './aliases/ManagedIdentityAliasesList';
import { GetConnections } from './ManagedIdentityList';
import ManagedIdentityTypeChip from './ManagedIdentityTypeChip';
import MoveManagedIdentityDialog from './MoveManagedIdentityDialog';
import ManagedIdentityRules from './rules/ManagedIdentityRules';
import ManagedIdentityWorkspaceList from './ManagedIdentityWorkspaceList';

interface Props {
    fragmentRef: ManagedIdentityDetailsFragment_group$key
}

const ISSUER = config.apiUrl;
const HOSTNAME = new URL(ISSUER).hostname;

// Template for Terraform HCL to configure managed identity access
const buildTerraformHCLTemplate = (apiUrl: string, managedIdentityId: string, audience: string) => {
    return `resource "kubernetes_cluster_role_binding_v1" "tharsis_managed_identity" {
  metadata {
    name = "tharsis-managed-identity-${managedIdentityId}"
  }
  
  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "cluster-admin"  # Adjust permissions as needed
  }
  
  subject {
    kind      = "User"
    name      = "${apiUrl}#${managedIdentityId}"
    api_group = "rbac.authorization.k8s.io"
  }
}

# Configure OIDC identity provider
resource "aws_eks_identity_provider_config" "tharsis" {
  cluster_name = "your-cluster-name"
  
  oidc {
    identity_provider_config_name = "tharsis"
    issuer_url                   = "${apiUrl}"
    client_id                    = "${audience}"
  }
}`;

};

const FieldLabel = styled(
    Typography
)(() => ({}));

const FieldValue = styled(
    Typography
)(({ theme }) => ({
    color: theme.palette.text.secondary,
    marginBottom: '16px',
    '&:last-child': {
        marginBottom: 0
    }
}));

function buildPolicy(role: string, sub: string): string {
    const rolePrefix = role.substring(0, role.indexOf(':role/'))
    return `{
        "Effect": "Allow",
        "Principal": {
            "Federated": "${rolePrefix}:oidc-provider/${HOSTNAME}"
        },
        "Action": "sts:AssumeRoleWithWebIdentity",
        "Condition": {
            "StringEquals": {
                "${HOSTNAME}:sub": "${sub}"
            }
        }
}`;
}

interface Props {
    fragmentRef: ManagedIdentityDetailsFragment_group$key
}

function ManagedIdentityDetails(props: Props) {
    const { id } = useParams();
    const [searchParams, setSearchParams] = useSearchParams();
    const { enqueueSnackbar } = useSnackbar();
    const navigate = useNavigate();
    const [menuAnchorEl, setMenuAnchorEl] = useState<Element | null>(null);
    const [showDeleteConfirmationDialog, setShowDeleteConfirmationDialog] = useState<boolean>(false);
    const [showMoveManagedIdentityDialog, setShowMoveManagedIdentityDialog] = useState<boolean>(false);

    const managedIdentityId = id as string;
    const tab = searchParams.get('tab') || 'details';

    const group = useFragment<ManagedIdentityDetailsFragment_group$key>(
        graphql`
        fragment ManagedIdentityDetailsFragment_group on Group
        {
          id
          fullPath
        }
        `, props.fragmentRef);

    const data = useLazyLoadQuery<ManagedIdentityDetailsQuery>(graphql`
        query ManagedIdentityDetailsQuery($id: String!, $first: Int!, $after: String, $last: Int, $before: String) {
            managedIdentity(id: $id) {
                id
                isAlias
                name
                description
                type
                data
                groupPath
                metadata {
                    trn
                }
                accessRules {
                    id
                    runStage
                    allowedUsers {
                        id
                        username
                        email
                    }
                    allowedTeams {
                        id
                        name
                    }
                    allowedServiceAccounts {
                        id
                        name
                        resourcePath
                    }
                }
                ...ManagedIdentityAliasesFragment_managedIdentity
                ...ManagedIdentityRulesFragment_managedIdentity
                ...MoveManagedIdentityDialogFragment_managedIdentity
            }
        }
    `, { id: managedIdentityId, first: INITIAL_ITEM_COUNT }, { fetchPolicy: 'store-and-network' });

    const [commitDelete, commitDeleteInFlight] = useMutation<ManagedIdentityDetailsDeleteMutation>(graphql`
        mutation ManagedIdentityDetailsDeleteMutation($input: DeleteManagedIdentityInput!, $connections: [ID!]!) {
            deleteManagedIdentity(input: $input) {
                managedIdentity {
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

    const onTabChange = (event: React.SyntheticEvent, newValue: string) => {
        searchParams.set('tab', newValue);
        setSearchParams(searchParams, { replace: true });
    };

    const onDeleteConfirmationDialogClosed = (confirm?: boolean) => {
        if (confirm) {
            commitDelete({
                variables: {
                    input: {
                        id: managedIdentityId
                    },
                    connections: GetConnections(group.id),
                },
                onCompleted: data => {
                    setShowDeleteConfirmationDialog(false);

                    if (data.deleteManagedIdentity.problems.length) {
                        enqueueSnackbar(data.deleteManagedIdentity.problems.map(problem => problem.message).join('; '), { variant: 'warning' });
                    } else {
                        navigate(`..`);
                    }
                },
                onError: error => {
                    setShowDeleteConfirmationDialog(false);
                    enqueueSnackbar(`Unexpected error occurred: ${error.message}`, { variant: 'error' });
                }
            });
        } else {
            setShowDeleteConfirmationDialog(false);
        }
    };

    const [commitDeleteAlias, commitDeleteAliasInFlight] = useMutation<ManagedIdentityDetailsDeleteAliasMutation>(graphql`
        mutation ManagedIdentityDetailsDeleteAliasMutation($input: DeleteManagedIdentityAliasInput!, $connections: [ID!]!) {
            deleteManagedIdentityAlias(input: $input){
                managedIdentity {
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

    const onDeleteAliasConfirmationDialogClosed = (confirm?: boolean) => {
        if (confirm) {
            commitDeleteAlias({
                variables: {
                    input: {
                        id: managedIdentityId
                    },
                    connections: GetConnections(group.id)
                },
                onCompleted: data => {
                    setShowDeleteConfirmationDialog(false);

                    if (data.deleteManagedIdentityAlias.problems.length) {
                        enqueueSnackbar(data.deleteManagedIdentityAlias.problems.map(problem => problem.message).join('; '), { variant: 'warning' });
                    } else {
                        navigate(`..`)
                    }
                },
                onError: error => {
                    setShowDeleteConfirmationDialog(false);
                    enqueueSnackbar(`Unexpected error occurred: ${error.message}`, { variant: 'error' });
                }
            });
        } else {
            setShowDeleteConfirmationDialog(false);
        }
    };

    const onOpenMenu = (event: React.MouseEvent<HTMLButtonElement>) => {
        setMenuAnchorEl(event.currentTarget);
    };

    const onMenuClose = () => {
        setMenuAnchorEl(null);
    };

    const onMenuAction = (actionCallback: () => void) => {
        setMenuAnchorEl(null);
        actionCallback();
    };

    if (data.managedIdentity && id && data.managedIdentity.groupPath === group.fullPath) {
        const payload = JSON.parse(atob(data.managedIdentity.data));
        return (
            <Box>
                <NamespaceBreadcrumbs
                    namespacePath={group.fullPath}
                    childRoutes={[
                        { title: "managed identities", path: 'managed_identities' },
                        { title: data.managedIdentity.name, path: id }
                    ]}
                />
                <Box display="flex" justifyContent="space-between" marginBottom={2}>
                    <Box>
                        <Box display="flex" alignItems="center">
                            <Typography variant="h5" sx={{ marginRight: 1 }}>{data.managedIdentity.name}</Typography>
                            <ManagedIdentityTypeChip mr={1} type={data.managedIdentity.type} />
                            {data.managedIdentity.isAlias && <Chip label="alias" color="secondary" size="small" />}

                        </Box>
                        <Typography color="textSecondary">{data.managedIdentity.description}</Typography>
                    </Box>
                    {!data.managedIdentity.isAlias && <Box>
                        <Stack direction="row" spacing={1}>
                            <TRNButton trn={data.managedIdentity.metadata.trn} />
                            <ButtonGroup variant="outlined" color="primary">
                                <Button onClick={() => navigate('edit')}>Edit</Button>
                                <Button
                                    color="primary"
                                    size="small"
                                    aria-label="more options menu"
                                    aria-haspopup="menu"
                                    onClick={onOpenMenu}
                                >
                                    <ArrowDropDownIcon fontSize="small" />
                                </Button>
                            </ButtonGroup>
                            <Menu
                                id="managed-identity-more-options-menu"
                                anchorEl={menuAnchorEl}
                                open={Boolean(menuAnchorEl)}
                                onClose={onMenuClose}
                                anchorOrigin={{
                                    vertical: 'bottom',
                                    horizontal: 'right',
                                }}
                                transformOrigin={{
                                    vertical: 'top',
                                    horizontal: 'right',
                                }}
                            >
                                <MenuItem onClick={() => onMenuAction(() => setShowMoveManagedIdentityDialog(true))}>
                                    Move Managed Identity
                                </MenuItem>
                                <MenuItem onClick={() => onMenuAction(() => setShowDeleteConfirmationDialog(true))}>
                                    Delete Managed Identity
                                </MenuItem>
                            </Menu>
                        </Stack>
                    </Box>}

                    {data.managedIdentity.isAlias && <Box>
                        <Stack direction="row" spacing={1}>
                            <TRNButton trn={data.managedIdentity.metadata.trn} />
                            <Button
                                variant="outlined"
                                color="error"
                                onClick={() => setShowDeleteConfirmationDialog(true)}
                            >Delete Alias</Button>
                        </Stack>
                    </Box>}
                </Box>
                <Box sx={{ display: "flex", justifyContent: "space-between", border: 1, borderTopLeftRadius: 4, borderTopRightRadius: 4, borderColor: 'divider' }}>
                    <Tabs value={tab} onChange={onTabChange}>
                        <Tab label="Details" value="details" />
                        <Tab label="Rules" value="rules" />
                        <Tab label="Workspaces" value="workspaces" />
                        {!data.managedIdentity.isAlias && <Tab label="Aliases" value="aliases" />}
                    </Tabs>
                </Box>
                <Box sx={{ border: 1, borderTop: 0, borderBottomLeftRadius: 4, borderBottomRightRadius: 4, borderColor: 'divider', padding: 2 }}>
                    {tab === 'details' && <Box>
                        {data.managedIdentity.type === 'aws_federated' && <Box>
                            <FieldLabel>IAM Role</FieldLabel>
                            <FieldValue>{payload.role}</FieldValue>
                            <FieldLabel>Audience</FieldLabel>
                            <FieldValue>aws</FieldValue>
                            <FieldLabel>IAM Trust Policy</FieldLabel>
                            <Typography color="textSecondary">Add the trust policy below to the IAM role in order to allow this managed identity to assume it.</Typography>
                            <SyntaxHighlighter wrapLongLines customStyle={{ fontSize: 14 }} language="json" style={prismTheme} children={buildPolicy(payload.role, payload.subject)} />
                        </Box>}
                        {data.managedIdentity.type === 'azure_federated' && <Box>
                            <FieldLabel>Issuer</FieldLabel>
                            <FieldValue>{ISSUER}</FieldValue>
                            <FieldLabel>Client ID</FieldLabel>
                            <FieldValue>{payload.clientId}</FieldValue>
                            <FieldLabel>Tenant ID</FieldLabel>
                            <FieldValue>{payload.tenantId}</FieldValue>
                            <FieldLabel>Audience</FieldLabel>
                            <FieldValue>azure</FieldValue>
                            <FieldLabel>Subject</FieldLabel>
                            <FieldValue>{payload.subject}</FieldValue>
                        </Box>}
                        {data.managedIdentity.type === 'tharsis_federated' && <Box>
                            <FieldLabel>Service Account</FieldLabel>
                            <FieldValue>{payload.serviceAccountPath}</FieldValue>
                            <FieldLabel>Use Service Account for Terraform CLI</FieldLabel>
                            <Typography color="textSecondary">{payload.useServiceAccountForTerraformCLI ? 'Yes' : 'No'}</Typography>
                            <FieldLabel marginTop={2}>Trusted Identity Provider</FieldLabel>
                            <Typography color="textSecondary">
                                Add the identity provider settings below to your service account to allow this managed identity to use it
                            </Typography>
                            <Paper sx={{ marginTop: 2, padding: 1 }}>
                                <Table>
                                    <TableHead>
                                        <TableRow>
                                            <TableCell>Issuer URL</TableCell>
                                            <TableCell>Bound Claims</TableCell>
                                        </TableRow>
                                    </TableHead>
                                    <TableBody>
                                        <TableRow
                                            sx={{ '&:last-child td, &:last-child th': { border: 0 }, height: 64 }}>
                                            <TableCell>{ISSUER}</TableCell>
                                            <TableCell>
                                                <Chip
                                                    size="small"
                                                    variant="outlined"
                                                    label={<React.Fragment>
                                                        <Typography variant="body2" component="span" sx={{ fontWeight: 'bold' }}>sub:</Typography>
                                                        <Typography variant="body2" component="span">{' ' + payload.subject}</Typography>
                                                    </React.Fragment>}
                                                />
                                            </TableCell>
                                        </TableRow>
                                    </TableBody>
                                </Table>
                            </Paper>
                        </Box>}
                        {data.managedIdentity.type === 'kubernetes_federated' && <Box>
                            <FieldLabel>Client ID</FieldLabel>
                            <FieldValue>{payload.audience}</FieldValue>
                            <FieldLabel>Subject Name</FieldLabel>
                            <FieldValue>{`${ISSUER}#${data.managedIdentity.id}`}</FieldValue>
                            <FieldLabel marginTop={2}>Terraform Configuration</FieldLabel>
                            <Typography color="textSecondary">
                                Use the Terraform configuration below to configure your EKS cluster OIDC identity provider and allow this managed identity access.
                            </Typography>
                            <SyntaxHighlighter wrapLongLines customStyle={{ fontSize: 14 }} language="hcl" style={prismTheme} children={buildTerraformHCLTemplate(config.apiUrl, data.managedIdentity.id, payload.audience)} />
                        </Box>}
                    </Box>}
                    {tab === 'rules' && <Box>
                        <ManagedIdentityRules
                            fragmentRef={data.managedIdentity}
                            groupPath={group.fullPath}
                        />
                    </Box>}
                    {tab === 'workspaces' && <Box>
                        <ManagedIdentityWorkspaceList managedIdentityId={data.managedIdentity.id} />
                    </Box>}
                    {tab === 'aliases' && <Box>
                        <ManagedIdentityAliases
                            fragmentRef={data.managedIdentity}
                        />
                    </Box>}
                </Box>
                {showDeleteConfirmationDialog && data.managedIdentity && (
                    <ConfirmationDialog
                        title={`Delete ${data.managedIdentity.isAlias ? 'Alias' : 'Managed Identity'}`}
                        confirmLabel="Delete"
                        confirmInProgress={data.managedIdentity.isAlias ? commitDeleteAliasInFlight : commitDeleteInFlight}
                        onConfirm={() => data.managedIdentity?.isAlias ? onDeleteAliasConfirmationDialogClosed(true) : onDeleteConfirmationDialogClosed(true)}
                        onClose={() => data.managedIdentity?.isAlias ? onDeleteAliasConfirmationDialogClosed() : onDeleteConfirmationDialogClosed()}
                    >
                        Are you sure you want to delete the {data.managedIdentity.isAlias ? 'alias' : 'managed identity'} <strong>{data.managedIdentity.name}</strong>?
                    </ConfirmationDialog>
                )}
                {showMoveManagedIdentityDialog && <MoveManagedIdentityDialog onClose={() => setShowMoveManagedIdentityDialog(false)} fragmentRef={data.managedIdentity} groupId={group.id} />}
            </Box>
        );
    } else {
        return <Box display="flex" justifyContent="center" marginTop={4}>
            <Typography color="textSecondary">Managed identity with ID {managedIdentityId} not found</Typography>
        </Box>;
    }
}

export default ManagedIdentityDetails
