import Chip from "@mui/material/Chip";
import React from 'react';

interface Props {
    type: string
    mr?: number
}

function ManagedIdentityTypeChip({ type, mr }: Props) {
    return (
        <React.Fragment>
            {type === 'aws_federated' && <Chip sx={{ color: '#FF9900', borderColor: '#FF9900', marginRight: mr }} variant="outlined" label={'aws'} size="small" />}
            {type === 'azure_federated' && <Chip sx={{ color: '#00a2ed', borderColor: '#00a2ed', marginRight: mr }} variant="outlined" label={'azure'} size="small" />}
            {type === 'tharsis_federated' && <Chip sx={{ color: '#4db6ac', borderColor: '#4db6ac', marginRight: mr }} variant="outlined" label={'tharsis'} size="small" />}
        </React.Fragment>
    )
}

export default ManagedIdentityTypeChip;
