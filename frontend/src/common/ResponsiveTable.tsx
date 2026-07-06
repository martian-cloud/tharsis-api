import {
    Box,
    Paper,
    Stack,
    Table,
    TableBody,
    TableCell,
    TableCellProps,
    TableContainer,
    TableHead,
    TableRow,
    Typography,
    useMediaQuery,
    useTheme,
} from '@mui/material';
import { Breakpoint } from '@mui/material/styles';
import { createContext, ReactNode, useContext } from 'react';

// CardModeContext is true when ResponsiveTable has switched to its mobile card layout, letting
// ResponsiveRow render cards instead of table rows without each caller re-checking the breakpoint.
const CardModeContext = createContext(false);

// useCardMode reports whether the surrounding ResponsiveTable is in mobile card mode, so callers can
// drop table-only affordances (e.g. spacer cells used for column alignment) when rendering as cards.
export const useCardMode = () => useContext(CardModeContext);

export interface ResponsiveColumn {
    label: ReactNode;
    align?: TableCellProps['align'];
}

interface ResponsiveTableProps {
    columns: ResponsiveColumn[];
    children: ReactNode;
    ariaLabel?: string;
    // minWidth keeps wide tables readable on desktop (they scroll within the container).
    minWidth?: number | string;
    // breakpoint below which rows render as stacked cards (default 'md').
    breakpoint?: Breakpoint;
}

// ResponsiveTable renders its rows as a normal table on desktop and as a stack of cards on small
// screens. Pair it with ResponsiveRow for the row content. Domain-agnostic — reusable anywhere a
// list/table needs to be mobile friendly.
export function ResponsiveTable({ columns, children, ariaLabel, minWidth, breakpoint = 'md' }: ResponsiveTableProps) {
    const theme = useTheme();
    const card = useMediaQuery(theme.breakpoints.down(breakpoint));

    if (card) {
        return (
            <CardModeContext.Provider value={true}>
                <Stack spacing={1}>{children}</Stack>
            </CardModeContext.Provider>
        );
    }

    return (
        <CardModeContext.Provider value={false}>
            <TableContainer>
                <Table aria-label={ariaLabel} sx={minWidth ? { minWidth } : undefined}>
                    <TableHead>
                        <TableRow>
                            {columns.map((column, index) => (
                                <TableCell key={index} align={column.align}>{column.label}</TableCell>
                            ))}
                        </TableRow>
                    </TableHead>
                    <TableBody>{children}</TableBody>
                </Table>
            </TableContainer>
        </CardModeContext.Provider>
    );
}

export interface ResponsiveCell {
    // label is shown as the field name in card mode; ignored for primary cells.
    label?: ReactNode;
    content: ReactNode;
    align?: TableCellProps['align'];
    // primary cells render prominently (no label) at the top of the card.
    primary?: boolean;
}

interface ResponsiveRowProps {
    cells: ResponsiveCell[];
}

// ResponsiveRow renders a single record as a table row (desktop) or a card (mobile), based on the
// surrounding ResponsiveTable.
export function ResponsiveRow({ cells }: ResponsiveRowProps) {
    const card = useContext(CardModeContext);

    if (card) {
        // Primary content goes top-left, label-less cells (actions) top-right, labeled fields stack below.
        const primaryCells = cells.filter((cell) => cell.primary);
        const actionCells = cells.filter((cell) => !cell.primary && !cell.label);
        const labeledCells = cells.filter((cell) => !cell.primary && cell.label);

        return (
            <Paper variant="outlined" sx={{ p: 2, overflowWrap: 'anywhere', backgroundColor: 'transparent' }}>
                <Stack spacing={1}>
                    {(primaryCells.length > 0 || actionCells.length > 0) && (
                        <Box display="flex" justifyContent="space-between" alignItems="flex-start" gap={2}>
                            <Box sx={{ minWidth: 0 }}>
                                {primaryCells.map((cell, index) => <Box key={index}>{cell.content}</Box>)}
                            </Box>
                            {actionCells.length > 0 && (
                                <Box sx={{ flexShrink: 0 }}>
                                    {actionCells.map((cell, index) => <Box key={index}>{cell.content}</Box>)}
                                </Box>
                            )}
                        </Box>
                    )}
                    {labeledCells.map((cell, index) => (
                        <Box key={index} display="flex" alignItems="center" gap={2} sx={{ minWidth: 0 }}>
                            <Typography variant="body2" color="textSecondary" sx={{ minWidth: 100, flexShrink: 0 }}>{cell.label}</Typography>
                            <Box sx={{ minWidth: 0 }}>{cell.content}</Box>
                        </Box>
                    ))}
                </Stack>
            </Paper>
        );
    }

    return (
        <TableRow sx={{ '&:last-child td, &:last-child th': { border: 0 } }}>
            {cells.map((cell, index) => (
                <TableCell key={index} align={cell.align}>{cell.content}</TableCell>
            ))}
        </TableRow>
    );
}
