import { Box, Button, TextField, Typography, useTheme } from '@mui/material';
import Alert from '@mui/material/Alert';
import Divider from '@mui/material/Divider';
import { nanoid } from 'nanoid';
import { MutationError } from '../../common/error';
import ServiceAccountFormTrustPolicy from './ServiceAccountFormTrustPolicy';

export interface FormData {
    name: string
    description: string
    oidcTrustPolicies: any[]
}

interface Props {
    data: FormData
    onChange: (data: FormData) => void
    editMode?: boolean
    error?: MutationError
}

function ServiceAccountForm({ data, onChange, editMode, error }: Props) {
    const theme = useTheme();

    const onNewIdentityProvider = () => {
        onChange({
            ...data,
            oidcTrustPolicies: [...data.oidcTrustPolicies, { issuer: '', boundClaims: [], _id: nanoid() }]
        });
    };

    const onDeleteIdentityProvider = (id: string) => {
        const index = data.oidcTrustPolicies.findIndex(trustPolicy => trustPolicy._id === id);
        if (index !== -1) {
            const oidcTrustPoliciesCopy = [...data.oidcTrustPolicies];
            oidcTrustPoliciesCopy.splice(index, 1)
            onChange({
                ...data,
                oidcTrustPolicies: oidcTrustPoliciesCopy
            });
        }
    };

    const onTrustPolicyChange = (trustPolicy: any) => {
        // Find trust policy
        const trustPolicyIndex = data.oidcTrustPolicies.findIndex(({ _id }) => _id === trustPolicy._id);
        if (trustPolicyIndex !== -1) {
            const trustPoliciesCopy = [...data.oidcTrustPolicies];
            trustPoliciesCopy[trustPolicyIndex] = trustPolicy;

            onChange({
                ...data,
                oidcTrustPolicies: trustPoliciesCopy
            });
        }
    };

    return (
        <Box>
            {error && <Alert sx={{ marginTop: 2 }} severity={error.severity}>
                {error.message}
            </Alert>}
            <Typography sx={{ marginTop: 2 }} variant="subtitle1" gutterBottom>Details</Typography>
            <Divider sx={{ opacity: 0.6 }} />
            <Box sx={{ my: 2 }}>
                <TextField
                    fullWidth
                    disabled={editMode}
                    size="small"
                    label="Name"
                    value={data.name}
                    onChange={event => onChange({ ...data, name: event.target.value })}
                />
                <TextField
                    size="small"
                    margin='normal'
                    fullWidth
                    label="Description"
                    value={data.description}
                    onChange={event => onChange({ ...data, description: event.target.value })}
                />
            </Box>
            <Box sx={{
                marginBottom: 1,
                display: 'flex',
                flexDirection: 'row',
                justifyContent: 'space-between',
                alignItems: 'center',
                [theme.breakpoints.down('md')]: {
                    flexDirection: 'column',
                    alignItems: 'flex-start',
                    '& > *': { marginBottom: 2 },
                }
            }}>
                <Box mb={1}>
                    <Typography variant="subtitle1">Trusted Identity Providers</Typography>
                    <Typography variant="caption" color="textSecondary">
                        Tokens issued by the following identity providers will be able to login to this service account provided that the bound claims match the token claims
                    </Typography>
                </Box>
                <Box>
                    <Button variant="outlined" size="small" sx={{ textTransform: 'none', minWidth: 200 }} color="secondary" onClick={onNewIdentityProvider}>
                        New Identity Provider
                    </Button>
                </Box>
            </Box>
            {data.oidcTrustPolicies.map(trustPolicy => (
                <ServiceAccountFormTrustPolicy
                    key={trustPolicy._id}
                    trustPolicy={trustPolicy}
                    onChange={onTrustPolicyChange}
                    onDelete={() => onDeleteIdentityProvider(trustPolicy._id)}
                />
            ))}
        </Box>
    );
}

export default ServiceAccountForm
