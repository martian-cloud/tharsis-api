import DeleteIcon from '@mui/icons-material/Delete';
import { IconButton, Paper } from '@mui/material';
import Box from '@mui/material/Box';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';

interface Props {
    policy: any;
    onChange: (policy: any) => void;
    onDelete: () => void;
}

function ManagedIdentityRuleModuleAttestationPolicy(props: Props) {
    const { policy, onChange, onDelete } = props;

    return (
        <Paper sx={{ marginBottom: 2, padding: 2, position: 'relative', background: 'inherit' }} variant="outlined">
            <IconButton
                sx={{ position: 'absolute', top: 2, right: 8 }}
                size="small"
                onClick={onDelete}
            >
                <DeleteIcon />
            </IconButton>
            <Box marginBottom={2}>
                <Typography gutterBottom>Public Key</Typography>
                <TextField
                    size="small"
                    rows={6}
                    multiline
                    margin='none'
                    placeholder="ECDSA public key used for attestation verification"
                    fullWidth
                    defaultValue={policy.publicKey}
                    onChange={event => onChange({ ...policy, publicKey: event.target.value })}
                />
            </Box>
            <Box>
                <Typography gutterBottom>Predicate Type (optional)</Typography>
                <TextField
                    size="small"
                    margin='none'
                    placeholder="Predicate Type"
                    fullWidth
                    defaultValue={policy.predicateType}
                    onChange={event => onChange({ ...policy, predicateType: event.target.value })}
                />
            </Box>
        </Paper>
    );
}

export default ManagedIdentityRuleModuleAttestationPolicy;
