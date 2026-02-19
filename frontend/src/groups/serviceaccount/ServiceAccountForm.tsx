import { Alert, Box, Button, Checkbox, Divider, FormControlLabel, Link, TextField, Typography } from '@mui/material';
import moment, { Moment } from 'moment';
import { nanoid } from 'nanoid';
import { useContext, useRef } from 'react';
import { ApiConfigContext } from '../../ApiConfigContext';
import { MutationError } from '../../common/error';
import ExpirationDateTimePicker from './ExpirationDateTimePicker';
import ServiceAccountFormTrustPolicy from './ServiceAccountFormTrustPolicy';

export const CLIENT_CREDENTIALS_DESCRIPTION = 'Allows authentication using a client ID and secret via the client credentials grant.';

export interface FormData {
    name: string
    description: string
    oidcTrustPolicies: any[]
    enableClientCredentials: boolean
    clientSecretExpiresAt?: Moment | null
}

interface Props {
    data: FormData
    onChange: (data: FormData) => void
    editMode?: boolean
    error?: MutationError
}

function ServiceAccountForm({ data, onChange, editMode, error }: Props) {
    const { serviceAccountClientSecretMaxExpirationDays: maxExpirationDays } = useContext(ApiConfigContext);
    const initialData = useRef(data).current;

    const onDeleteIdentityProvider = (id: string) => {
        onChange({
            ...data,
            oidcTrustPolicies: data.oidcTrustPolicies.filter(p => p._id !== id)
        });
    };

    const onTrustPolicyChange = (trustPolicy: any) => {
        onChange({
            ...data,
            oidcTrustPolicies: data.oidcTrustPolicies.map(p => p._id === trustPolicy._id ? trustPolicy : p)
        });
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
            <Typography sx={{ marginTop: 3 }} variant="subtitle1" gutterBottom>OIDC Authentication</Typography>
            <Divider sx={{ opacity: 0.6 }} />
            <Box sx={{ my: 1 }}>
                <Typography variant="caption" color="textSecondary">
                    Tokens issued by trusted identity providers can authenticate to this service account if the bound claims match the token claims.
                </Typography>
            </Box>
            {data.oidcTrustPolicies.map(trustPolicy => (
                <ServiceAccountFormTrustPolicy
                    key={trustPolicy._id}
                    trustPolicy={trustPolicy}
                    onChange={onTrustPolicyChange}
                    onDelete={() => onDeleteIdentityProvider(trustPolicy._id)}
                />
            ))}
            <Button
                variant="outlined"
                size="small"
                sx={{ textTransform: 'none', minWidth: 200, mt: 1, mb: 1 }}
                color="secondary"
                onClick={() => onChange({ ...data, oidcTrustPolicies: [...data.oidcTrustPolicies, { issuer: '', boundClaims: [], _id: nanoid() }] })}
            >
                New Identity Provider
            </Button>
            <Typography sx={{ marginTop: 3 }} variant="subtitle1" gutterBottom>Client Credentials Authentication</Typography>
            <Divider sx={{ opacity: 0.6 }} />
            <Box sx={{ my: 1 }}>
                <Typography variant="caption" color="textSecondary">
                    {CLIENT_CREDENTIALS_DESCRIPTION}{' '}
                    <Link
                        href="https://auth0.com/docs/get-started/authentication-and-authorization-flow/client-credentials-flow"
                        target="_blank"
                        rel="noopener noreferrer"
                        underline="hover"
                    >
                        Learn more
                    </Link>
                </Typography>
                <FormControlLabel
                    sx={{ display: 'block', mt: 1 }}
                    control={
                        <Checkbox
                            checked={data.enableClientCredentials}
                            onChange={event => onChange({
                                ...data,
                                enableClientCredentials: event.target.checked,
                                clientSecretExpiresAt: event.target.checked ? moment().add(maxExpirationDays, 'days') : null
                            })}
                        />
                    }
                    label="Enable"
                />
            </Box>
            {data.enableClientCredentials && (
                <ExpirationDateTimePicker
                    value={data.clientSecretExpiresAt ?? null}
                    onChange={value => onChange({ ...data, clientSecretExpiresAt: value })}
                    maxExpirationDays={maxExpirationDays}
                    disabled={initialData.enableClientCredentials}
                />
            )}
        </Box>
    );
}

export default ServiceAccountForm
