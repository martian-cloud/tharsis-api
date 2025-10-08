import { TableCell, TableRow, Typography } from '@mui/material'
import Tooltip from '@mui/material/Tooltip'
import Box from '@mui/material/Box'
import React from 'react'
import Link from '../../routes/Link'
import Gravatar from '../../common/Gravatar'
import { useFragment } from 'react-relay/hooks'
import { StateVersionListItemFragment_stateVersion$key } from './__generated__/StateVersionListItemFragment_stateVersion.graphql'
import graphql from 'babel-plugin-relay/macro'

interface Props {
    stateVersionKey: StateVersionListItemFragment_stateVersion$key
    workspacePath: string
}

function StateVersionListItem(props: Props) {
    const { stateVersionKey, workspacePath } = props

    const data = useFragment<StateVersionListItemFragment_stateVersion$key>(graphql`
        fragment StateVersionListItemFragment_stateVersion on StateVersion {
            id
            createdBy
            metadata {
                createdAt
                trn
            }
            run {
                id
                createdBy
            }
        } `, stateVersionKey)

    const stateVersionValue = data.run ? data.run.createdBy : data.createdBy

    return (
        <TableRow
            sx={{ '&:last-child td, &:last-child th': { border: 0 } }}
        >
            <TableCell>
                <Link color="inherit" to={`/groups/${workspacePath}/-/state_versions/${data.id}`}>{data.id.substring(0, 8)}...</Link>
            </TableCell>
            <TableCell>
                {data.run ? <Link color="inherit" to={`/groups/${workspacePath}/-/runs/${data.run.id}`}>{data.run.id.substring(0, 8)}...</Link> : <Typography>created manually</Typography>}
            </TableCell>
            <TableCell>
                <Typography>{data.metadata.createdAt}</Typography>
            </TableCell>
            <TableCell>
                <Tooltip title={stateVersionValue}>
                    <Box>
                        <Gravatar width={24} height={24} email={stateVersionValue} />
                    </Box>
                </Tooltip>
            </TableCell>
        </TableRow>
    );
}

export default StateVersionListItem
