import { Box, Button, Stack, TableCell, TableRow, Tooltip, Typography } from "@mui/material"
import DeleteIcon from '@mui/icons-material/CloseOutlined';
import { useFragment } from "react-relay"
import moment from 'moment';
import graphql from 'babel-plugin-relay/macro'
import Gravatar from '../common/Gravatar'
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
            createdBy
        }
    `, fragmentRef);

    return (
        <TableRow sx={{ '&:last-child td, &:last-child th': { border: 0 } }}>
            <TableCell>{data.id.substring(0, 8)}...</TableCell>
            <TableCell sx={{ wordWrap: "break-word" }}>{data.description}</TableCell>
            <TableCell>
                <Typography sx={{ wordWrap: "break-word" }} variant="body2">{data.predicateType}</Typography>
            </TableCell>
            <TableCell>
                <Box display="flex" alignItems="center">
                    {moment(data.metadata.createdAt as moment.MomentInput).fromNow()} by
                    <Tooltip sx={{ ml: 1 }} title={data.createdBy}>
                        <Box>
                            <Gravatar width={20} height={20} email={data.createdBy} />
                        </Box>
                    </Tooltip>
                </Box>
            </TableCell>
            <TableCell>
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
            </TableCell>
        </TableRow>
    );
}

export default TerraformModuleVersionAttestListItem
