import { GitHub } from "@mui/icons-material";
import { Gitlab } from "mdi-material-ui";
import React from 'react';

interface Props {
    type: string
}

function VCSProviderTypeChip({ type }: Props) {
    return (
        <React.Fragment>
            {type === 'github' && <GitHub/>}
            {type === 'gitlab' && <Gitlab/>}
        </React.Fragment>
    )
}

export default VCSProviderTypeChip
