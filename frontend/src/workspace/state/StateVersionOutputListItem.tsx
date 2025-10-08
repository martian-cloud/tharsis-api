import TableRow from '@mui/material/TableRow';
import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import DataTableCell from '../../common/DataTableCell';
import { StateVersionOutputListItemFragment_output$key } from './__generated__/StateVersionOutputListItemFragment_output.graphql';

interface Props {
    fragmentRef: StateVersionOutputListItemFragment_output$key;
    showValues: boolean;
}

function StateVersionOutputListItem(props: Props) {
    const { fragmentRef, showValues } = props;
    const data = useFragment<StateVersionOutputListItemFragment_output$key>(
        graphql`
        fragment StateVersionOutputListItemFragment_output on StateVersionOutput
        {
            name
            value
            type
            sensitive
        }
      `, fragmentRef);

    const value = data.type === '"string"' ? data.value.slice(1, -1) : data.value;

    return (
        <TableRow
            sx={{ '&:last-child td, &:last-child th': { border: 0 } }}
        >
            <DataTableCell sx={{ wordBreak: 'break-all' }}>
                {data.name}
            </DataTableCell>
            <DataTableCell sx={{ wordBreak: 'break-all' }} mask={!showValues && data.sensitive} >
                {value}
            </DataTableCell>
        </TableRow>
    );
}

export default StateVersionOutputListItem;
