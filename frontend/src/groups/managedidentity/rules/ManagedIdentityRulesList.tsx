import DeleteIcon from '@mui/icons-material/CloseOutlined';
import EditIcon from '@mui/icons-material/EditOutlined';
import { Avatar, Box, Button, Chip, styled, Tooltip, Typography } from '@mui/material';
import { Fragment, useMemo } from 'react';
import Gravatar from '../../../common/Gravatar';
import { ResponsiveRow, ResponsiveTable } from '../../../common/ResponsiveTable';

const AvatarContainer = styled(
    Box
)(() => ({
    display: 'flex',
    flexWrap: 'wrap',
    margin: '0 -4px',
    '& > *': {
        margin: '4px'
    }
}));

const StyledAvatar = styled(
    Avatar
)(() => ({
    width: 24,
    height: 24,
    marginRight: 1,
    backgroundColor: 'avatar.default',
}));

const ACCESS_RULE_TYPE_LABELS = {
    eligible_principals: 'Eligible Principals',
    module_attestation: 'Module Attestation'
} as any;

function buildPrincipals(rule: any) {
    return [
        ...rule.allowedUsers.map((user: any) => ({ id: user.id, type: 'user', label: user.email, tooltip: user.email, name: user.username })),
        ...rule.allowedTeams.map((team: any) => ({ id: team.id, type: 'team', label: team.name[0].toUpperCase(), tooltip: team.name, name: team.name })),
        ...rule.allowedServiceAccounts.map((sa: any) => ({ id: sa.id, type: 'serviceaccount', label: sa.name[0].toUpperCase(), tooltip: sa.resourcePath, name: sa.resourcePath }))
    ];
}

interface Props {
    accessRules: any;
    isAlias?: boolean;
    onEdit: (rule: any) => void;
    onDelete: (rule: any) => void;
}

function ManagedIdentityRulesList(props: Props) {
    const { accessRules, isAlias, onEdit, onDelete } = props;

    const rows = useMemo(() => (accessRules ?? []).map((rule: any) => ({
        type: rule.type,
        principals: buildPrincipals(rule),
        rule: rule
    })), [accessRules]);

    return (
        <ResponsiveTable
            ariaLabel="managed identity rules"
            columns={isAlias
                ? [{ label: 'Type' }, { label: 'Policy' }, { label: 'Run Stage' }]
                : [{ label: 'Type' }, { label: 'Policy' }, { label: 'Run Stage' }, { label: '', align: 'right' }]}
        >
            {rows.map((row: any) => (
                <ResponsiveRow key={row.rule.id} cells={[
                    { primary: true, content: <Typography variant="body2">{ACCESS_RULE_TYPE_LABELS[row.type]}</Typography> },
                    {
                        label: 'Policy', content: <>
                            {row.type === 'eligible_principals' && <Fragment>
                                {row.principals && row.principals.length === 0 && <Typography variant="body2" fontWeight={500}>
                                    No principals are permitted
                                </Typography>}
                                {row.principals && row.principals.length > 0 && <Box>
                                    <Typography gutterBottom variant="body2" fontWeight={500}>Only the following principals are permitted:</Typography>
                                    <AvatarContainer>
                                        {row.principals.map(((rule: any) => (
                                            <Tooltip key={rule.id} title={rule.tooltip}>
                                                <Box>
                                                    {rule.type === 'user' && <Gravatar width={24} height={24} email={rule.label} />}
                                                    {rule.type !== 'user' && <StyledAvatar>{rule.label}</StyledAvatar>}
                                                </Box>
                                            </Tooltip>
                                        )))}
                                    </AvatarContainer>
                                </Box>}
                            </Fragment>}
                            {row.type === 'module_attestation' && <Typography variant="body2" fontWeight={500}>
                                Only root modules with the required attestations are permitted
                            </Typography>}
                        </>
                    },
                    { label: 'Run Stage', content: <Chip size="small" label={`${row.rule.runStage[0].toUpperCase()}${row.rule.runStage.slice(1)}`} /> },
                    ...(!isAlias ? [{
                        align: 'right' as const, content: <>
                            <Button sx={{ marginRight: 1, minWidth: 40, padding: '2px' }} size="small" color="info" variant="outlined" onClick={() => onEdit(row.rule)}>
                                <EditIcon />
                            </Button>
                            <Button sx={{ minWidth: 40, padding: '2px' }} size="small" color="info" variant="outlined" onClick={() => onDelete(row.rule)}>
                                <DeleteIcon />
                            </Button>
                        </>
                    }] : [])
                ]} />
            ))}
        </ResponsiveTable>
    );
}

export default ManagedIdentityRulesList;
