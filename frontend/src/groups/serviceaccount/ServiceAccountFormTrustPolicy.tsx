import DeleteIcon from '@mui/icons-material/Delete';
import InfoIcon from '@mui/icons-material/Info';
import { Box, Button, Checkbox, Chip, FormControlLabel, IconButton, Paper, TextField, Tooltip, Typography } from '@mui/material';
import React, { useState } from 'react';
import EditClaimDialog from './EditClaimDialog';

interface Props {
    trustPolicy: any;
    onChange: (trustPolicy: any) => void;
    onDelete: () => void;
}

function ServiceAccountFormTrustPolicy(props: Props) {
    const { trustPolicy, onChange, onDelete } = props;

    const [claimToEdit, setClaimToEdit] = useState<any>(null);
    const [editClaimError, setEditClaimError] = useState('');

    const onDeleteClaim = (claim: any) => {
        const trustPolicyCopy = { ...trustPolicy };

        const claimsCopy = [...trustPolicy.boundClaims];
        const claimIndex = claimsCopy.findIndex(({ name }) => name === claim.name);
        if (claimIndex !== -1) {
            claimsCopy.splice(claimIndex, 1)
        }

        trustPolicyCopy.boundClaims = claimsCopy;

        onChange(trustPolicyCopy);
    };

    const onEditClaimDialogClosed = (claim?: any) => {
        setEditClaimError('');

        if (claim) {
            const trustPolicyCopy = { ...trustPolicy };

            // Check if a claim already exists with the updated claim's name
            if (claimToEdit.name !== claim.name) {
                const match = trustPolicy.boundClaims.find((c: any) => claim.name === c.name);
                if (match) {
                    setEditClaimError(`Claim with name ${claim.name} already exists`);
                    return;
                }
            }

            const claimsCopy = [...trustPolicy.boundClaims];
            const claimIndex = claimsCopy.findIndex(({ name }) => name === claimToEdit.name);
            if (claimIndex !== -1) {
                // Replace existing claim
                claimsCopy[claimIndex] = claim;
            } else {
                // Add new claim
                claimsCopy.push(claim);
            }

            trustPolicyCopy.boundClaims = claimsCopy;

            onChange(trustPolicyCopy);
        }

        setClaimToEdit(null);
    };

    return (
        <Paper key={trustPolicy._id} sx={{ marginBottom: 2, padding: 2, position: 'relative' }} variant="outlined">
            <IconButton
                sx={{ position: 'absolute', top: 2, right: 8 }}
                size="small"
                onClick={onDelete}
            >
                <DeleteIcon />
            </IconButton>
            <Box mb={3}>
                <Typography gutterBottom>Issuer URL</Typography>
                <TextField
                    size="small"
                    margin='none'
                    placeholder="Issuer URL for identity provider"
                    fullWidth
                    defaultValue={trustPolicy.issuer}
                    onChange={event => onChange({ ...trustPolicy, issuer: event.target.value })}
                />
            </Box>
            <Box mb={3}>
                <Typography gutterBottom>Bound Claims</Typography>
                <Box display="flex" flexWrap="wrap" sx={{
                    margin: '0 -4px',
                    '& > *': {
                        margin: '4px'
                    },
                }}>
                    {trustPolicy.boundClaims.map((claim: any) => (
                        <Box key={claim.name}>
                            <Chip
                                component="span"
                                variant="outlined"
                                label={<React.Fragment>
                                    <Typography variant="body2" component="span" sx={{ fontWeight: 'bold' }}>{claim.name}:</Typography>
                                    <Typography variant="body2" component="span">{' ' + claim.value}</Typography>
                                </React.Fragment>}
                                onClick={() => setClaimToEdit(claim)}
                                onDelete={() => onDeleteClaim(claim)}
                            />
                        </Box>
                    ))}
                    <Button
                        sx={{ textTransform: 'none' }}
                        color="secondary"
                        onClick={() => setClaimToEdit({ name: '', value: '' })}
                    >
                        New Claim
                    </Button>
                </Box>
            </Box>
            <FormControlLabel
                control={<Checkbox
                    color="secondary"
                    checked={trustPolicy.boundClaimsType === 'GLOB'}
                    onChange={event => onChange({ ...trustPolicy, boundClaimsType: event.target.checked ? 'GLOB' : 'STRING' })}
                />}
                label={
                    <Box display="flex" alignItems="center">
                        <Typography variant="body2">Enable wildcard matching for bound claims</Typography>
                        <Tooltip title="When enabled, wildcard characters can be used in claim values to match any string">
                            <InfoIcon sx={{
                                width: 16,
                                height: 16,
                                marginLeft: '4px',
                                verticalAlign: 'middle',
                                opacity: '20%',
                                transition: 'ease',
                                transitionDuration: '300ms',
                                ":hover": {
                                    opacity: '100%'
                                }
                            }} />
                        </Tooltip>
                    </Box>
                }
            />
            {claimToEdit && <EditClaimDialog
                claim={claimToEdit}
                error={editClaimError}
                editMode={claimToEdit.name !== ''}
                onClose={onEditClaimDialogClosed}
            />}
        </Paper>
    );
}

export default ServiceAccountFormTrustPolicy;
