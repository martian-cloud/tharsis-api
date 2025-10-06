import Chip from '@mui/material/Chip';
import TableCell from '@mui/material/TableCell';
import TableRow from '@mui/material/TableRow';
import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { StateVersionResourceListItemFragment_resource$key } from './__generated__/StateVersionResourceListItemFragment_resource.graphql';

interface Props {
    fragmentRef: StateVersionResourceListItemFragment_resource$key
}

function StateVersionResourceListItem(props: Props) {
    const { fragmentRef } = props;
    const data = useFragment<StateVersionResourceListItemFragment_resource$key>(
        graphql`
        fragment StateVersionResourceListItemFragment_resource on StateVersionResource
        {
            name
            type
            provider
            mode
            module
        }
      `, fragmentRef);

    return (
        <TableRow
            sx={{ '&:last-child td, &:last-child th': { border: 0 } }}
        >
            <TableCell sx={{ wordBreak: 'break-all' }}>
                {data.name}
            </TableCell>
            <TableCell sx={{ wordBreak: 'break-all' }}>
                {data.type}
                {data.mode === 'data' && <Chip size="small" label='datasource' sx={{ marginLeft: 1}} />}
            </TableCell>
            <TableCell sx={{ wordBreak: 'break-all' }}>
                {data.provider}
            </TableCell>
            <TableCell sx={{ wordBreak: 'break-all' }}>
                {data.module}
            </TableCell>
        </TableRow>
    );
}

export default StateVersionResourceListItem;