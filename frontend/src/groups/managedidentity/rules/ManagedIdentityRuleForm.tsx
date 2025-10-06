import DeleteIcon from '@mui/icons-material/Delete';
import { Alert, Avatar, Box, Button, IconButton, List, ListItem, ListItemText, Stack, styled, Typography } from '@mui/material';
import teal from '@mui/material/colors/teal';
import MenuItem from '@mui/material/MenuItem';
import Select, { SelectChangeEvent } from '@mui/material/Select';
import { useTheme } from '@mui/material/styles';
import { nanoid } from 'nanoid';
import { Fragment } from 'react';
import { MutationError } from '../../../common/error';
import Gravatar from '../../../common/Gravatar';
import PanelButton from '../../../common/PanelButton';
import ManagedIdentityRuleModuleAttestationPolicy from './ManagedIdentityRuleModuleAttestationPolicy';
import PrincipalAutocomplete, { Option, ServiceAccountOption, TeamOption, UserOption } from './PrincipalAutocomplete';

const StyledAvatar = styled(
    Avatar
)(() => ({
    width: 24,
    height: 24,
    marginRight: 2,
    backgroundColor: teal[200],
}));

const RUN_STAGES = [
    { name: 'plan', label: 'Plan' },
    { name: 'apply', label: 'Apply' }
];

interface Props {
    groupPath: string;
    rule: any;
    onChange: (rule: any) => void
    editMode?: boolean
    error?: MutationError
}

function ManagedIdentityRuleForm(props: Props) {
    const { groupPath, rule, onChange, editMode, error } = props

    const theme = useTheme();

    const handleDelete = (principal: any) => {
        function deletePrincipal(field: string) {
            const copy = [...rule[field]];
            const index = copy.findIndex(item => item.id === principal.id);
            if (index !== -1) {
                copy.splice(index, 1);
                onChange({ ...rule, [field]: copy });
            }
        }

        switch (principal.type) {
            case 'user': {
                deletePrincipal('allowedUsers');
                break;
            }
            case 'team': {
                deletePrincipal('allowedTeams');
                break;
            }
            case 'serviceaccount': {
                deletePrincipal('allowedServiceAccounts');
                break;
            }
        }
    }

    const onSelected = (value: Option | null) => {
        if (value) {
            switch (value.type) {
                case 'user': {
                    const user = value as UserOption;
                    onChange({ ...rule, allowedUsers: [...rule.allowedUsers, { id: user.id, email: user.email, username: user.username }] });
                    break;
                }
                case 'team': {
                    const team = value as TeamOption;
                    onChange({ ...rule, allowedTeams: [...rule.allowedTeams, { id: team.id, name: team.name }] });
                    break;
                }
                case 'serviceaccount': {
                    const sa = value as ServiceAccountOption;
                    onChange({ ...rule, allowedServiceAccounts: [...rule.allowedServiceAccounts, { id: sa.id, name: sa.name, resourcePath: sa.resourcePath }] });
                    break;
                }
            }
        }
    };

    const onTypeChange = (type: string) => {
        if (!editMode) {
            onChange({
                ...rule,
                type,
                allowedUsers: [],
                allowedServiceAccount: [],
                allowedTeams: [],
                moduleAttestationPolicies: [{ publicKey: '', predicateType: '', _id: nanoid() }]
            });
        }
    };

    const onRunStageChange = (event: SelectChangeEvent) => {
        onChange({ ...rule, runStage: event.target.value });
    };

    const onModuleAttestationPolicyChange = (policy: any) => {
        // Find the policy
        const index = rule.moduleAttestationPolicies.findIndex(({ _id }: any) => _id === policy._id);
        if (index !== -1) {
            const moduleAttestationPoliciesCopy = [...rule.moduleAttestationPolicies];
            moduleAttestationPoliciesCopy[index] = policy;

            onChange({
                ...rule,
                moduleAttestationPolicies: moduleAttestationPoliciesCopy
            });
        }
    };

    const onDeleteModuleAttestationPolicy = (id: string) => {
        const index = rule.moduleAttestationPolicies.findIndex((policy: any) => policy._id === id);
        if (index !== -1) {
            const moduleAttestationPoliciesCopy = [...rule.moduleAttestationPolicies];
            moduleAttestationPoliciesCopy.splice(index, 1)
            onChange({
                ...rule,
                moduleAttestationPolicies: moduleAttestationPoliciesCopy
            });
        }
    };

    const onNewModuleAttestationPolicy = () => {
        onChange({ ...rule, moduleAttestationPolicies: [...rule.moduleAttestationPolicies, { publicKey: '', predicateType: '', _id: nanoid() }] })
    };

    const principals = [
        ...rule.allowedUsers.map((user: any) => ({ id: user.id, type: 'user', label: user.email, tooltip: user.email, name: user.username })),
        ...rule.allowedTeams.map((team: any) => ({ id: team.id, type: 'team', label: team.name[0].toUpperCase(), tooltip: team.name, name: team.name })),
        ...rule.allowedServiceAccounts.map((sa: any) => ({ id: sa.id, type: 'serviceaccount', label: sa.name[0].toUpperCase(), tooltip: sa.resourcePath, name: sa.resourcePath }))
    ];

    const selectedIds = principals.reduce((accumulator: Set<string>, item: any) => {
        accumulator.add(item.id);
        return accumulator;
    }, new Set());

    return (
        <Box>
            {error && <Alert sx={{ marginTop: 2, marginBottom: 2 }} severity={error.severity}>
                {error.message}
            </Alert>}
            <Box marginBottom={2}>
                <Typography variant="subtitle1" gutterBottom>Rule Type</Typography>
                <Stack marginTop={2} direction="row" spacing={2}>
                    <PanelButton
                        disabled={editMode}
                        selected={rule.type === 'eligible_principals'}
                        onClick={() => onTypeChange('eligible_principals')}
                    >
                        <Typography variant="subtitle1">Eligible Principals</Typography>
                        <Typography variant="caption" align="center">
                            Restricts which users, service accounts, and teams are allowed to use this managed identity
                        </Typography>
                    </PanelButton>
                    <PanelButton
                        disabled={editMode}
                        selected={rule.type === 'module_attestation'}
                        onClick={() => onTypeChange('module_attestation')}
                    >
                        <Typography variant="subtitle1">Module Attestation</Typography>
                        <Typography variant="caption" align="center">
                            Only root modules that have the required attestations can be used with this managed identity
                        </Typography>
                    </PanelButton>
                </Stack>
            </Box>
            {rule.type && <Box marginBottom={2}>
                <Typography sx={{ mb: 1 }} variant="body1">Run Stage</Typography>
                <Select
                    disabled={editMode}
                    sx={{ minWidth: 120 }}
                    size="small"
                    value={rule.runStage}
                    onChange={onRunStageChange}
                >
                    {RUN_STAGES.map(stage => <MenuItem key={stage.name} value={stage.name}>{stage.label}</MenuItem>)}
                </Select>
            </Box>}
            {rule.type === 'eligible_principals' && <Fragment>
                <Typography sx={{ mb: 1 }} variant="body1">Add Principals</Typography>
                <Box sx={{ border: `1px solid ${theme.palette.divider}`, borderRadius: '4px' }} marginBottom={4} padding={2}>
                    <Box sx={{ marginBottom: 2 }}>
                        <PrincipalAutocomplete
                            groupPath={groupPath}
                            onSelected={onSelected}
                            filterOptions={(options: Option[]) => options.filter(option => !selectedIds.has(option.id))}
                        />
                    </Box>
                    <Typography color="textSecondary">
                        {principals.length} principal{principals.length === 1 ? '' : 's'} selected
                    </Typography>
                    <List dense>
                        {principals?.map((pr: any) => (
                            <ListItem
                                disableGutters
                                secondaryAction={<IconButton onClick={() => handleDelete(pr)}>
                                    <DeleteIcon />
                                </IconButton>}
                                key={pr.id}>
                                {pr.type === 'user' && <Gravatar sx={{ marginRight: 1 }} width={24} height={24} email={pr.label} />}
                                {pr.type !== 'user' &&
                                    <StyledAvatar sx={{ marginRight: 1 }}>{pr.label}</StyledAvatar>}
                                <ListItemText primary={pr.name} primaryTypographyProps={{ noWrap: true }} />
                            </ListItem>))}
                    </List>
                </Box>
            </Fragment>}
            {rule.type === 'module_attestation' && <Box>
                <Box marginBottom={2}>
                    <Typography variant="subtitle1">Module Attestation Policies</Typography>
                    <Typography variant="caption" color="textSecondary">
                        All of the module attestation policies must be satisfied
                    </Typography>
                </Box>
                {rule.moduleAttestationPolicies?.map((policy: any) => <ManagedIdentityRuleModuleAttestationPolicy
                    key={policy._id}
                    policy={policy}
                    onChange={onModuleAttestationPolicyChange}
                    onDelete={() => onDeleteModuleAttestationPolicy(policy._id)}
                />)}
                <Box>
                    <Button variant="outlined" size="small" sx={{ textTransform: 'none', minWidth: 200 }} color="secondary" onClick={onNewModuleAttestationPolicy}>
                        Add Policy
                    </Button>
                </Box>
            </Box>}
        </Box>
    )
}

export default ManagedIdentityRuleForm;
