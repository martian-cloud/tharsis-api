import * as React from 'react';
import TableCell, { TableCellProps } from '@mui/material/TableCell';

export const monoFontFamily = 'ui-monospace,SFMono-Regular,SF Mono,Menlo,Consolas,Liberation Mono,monospace !important';

// Placeholder shown in place of hidden/sensitive values across the variable and output tables.
export const MASKED_VALUE = '************';

interface DataTableCellProps extends TableCellProps {
    mask?: boolean;
}

function DataTableCell(props: DataTableCellProps) {
    const { sx, mask, ...other } = props;

    return (
        <TableCell  {...other} sx={{ ...sx, fontFamily: mask ? undefined : monoFontFamily }}>
            {mask ? MASKED_VALUE : props.children}
        </TableCell>
    );
}

export default DataTableCell;
