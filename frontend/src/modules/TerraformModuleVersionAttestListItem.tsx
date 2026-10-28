import { Box, Button, Stack, Typography } from "@mui/material"
import DeleteIcon from '@mui/icons-material/CloseOutlined';
import { useFragment } from "react-relay"
import graphql from 'babel-plugin-relay/macro'
import ActorDisplay from '../common/ActorDisplay'
import { ResponsiveRow } from '../common/ResponsiveTable'
import Timestamp from '../common/Timestamp'
import { TerraformModuleVersionAttestListItemFragment_module$key } from "./__generated__/TerraformModuleVersionAttestListItemFragment_module.graphql"

interface Props {
    fragmentRef: TerraformModuleVersionAttestListItemFragment_module$key
    onOpenDataDialog: () => void
    onOpenDeleteDialog: () => void
}

function TerraformModuleVersionAttestListItem({ fragmentRef, onOpenDataDialog, onOpenDeleteDialog }: Props) {

    const data = useFragment<TerraformModuleVersionAttestListItemFragment_module$key>(
        graphql`
        fragment TerraformModuleVersionAttestListItemFragment_module on TerraformModuleAttestation
        {
            id
            description
            predicateType
            data
            metadata {
                createdAt
            }
            createdByActor {
                ...ActorDisplayFragment_actor
            }
        }
    `, fragmentRef);

    const created = (
        <Box display="flex" alignItems="center" gap={0.5}>
            <Timestamp variant="body2" timestamp={data.metadata.createdAt as string} />
            <Typography variant="body2">by</Typography>
            <ActorDisplay fragmentRef={data.createdByActor} variant="avatar" size={20} />
        </Box>
    );

    const actions = (
        <Stack direction="row" spacing={1}>
            <Button
                size="small"
                color="info"
                variant="outlined"
                onClick={onOpenDataDialog}>View Data
            </Button>
            <Button
                sx={{ minWidth: 40, padding: '2px' }}
                size="small"
                color="info"
                variant="outlined"
                onClick={onOpenDeleteDialog}><DeleteIcon />
            </Button>
        </Stack>
    );

    return (
        <ResponsiveRow cells={[
            { primary: true, content: <Typography fontWeight={500}>{data.id.substring(0, 8)}...</Typography> },
            { label: 'Description', content: <Typography variant="body2">{data.description}</Typography> },
            { label: 'Predicate Type', content: <Typography variant="body2">{data.predicateType}</Typography> },
            { label: 'Created', content: created },
            { align: 'right', content: actions },
        ]} />
    );
}

export default TerraformModuleVersionAttestListItem
