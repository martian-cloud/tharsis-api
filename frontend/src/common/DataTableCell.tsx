import * as React from 'react';
import TableCell, { TableCellProps } from '@mui/material/TableCell';

const monoFontFamily = 'ui-monospace,SFMono-Regular,SF Mono,Menlo,Consolas,Liberation Mono,monospace !important';

interface DataTableCellProps extends TableCellProps {
    mask?: boolean;
}

function DataTableCell(props: DataTableCellProps) {
    const { sx, mask, ...other } = props;

    return (
        <TableCell  {...other} sx={{ ...sx, fontFamily: mask ? undefined : monoFontFamily }}>
            {mask ? '************' : props.children}
        </TableCell>
    );
}

export default DataTableCell;
