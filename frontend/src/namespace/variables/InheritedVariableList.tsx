import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Typography from '@mui/material/Typography';
import React from 'react';
import InheritedVariableListItem from './InheritedVariableListItem';

interface Props {
    variables: any[]
    showValues: boolean
    variableCategory: string
}

function InheritedVariableList(props: Props) {
    const { variables, showValues, variableCategory } = props;

    return (
        <TableContainer>
            <Table sx={{ tableLayout: 'fixed' }}>
                <TableHead>
                    <TableRow>
                        <TableCell>
                            <Typography color="textSecondary">Key</Typography>
                        </TableCell>
                        <TableCell>
                            <Typography color="textSecondary">Value</Typography>
                        </TableCell>
                        {variableCategory === 'terraform' && <TableCell>
                            <Typography color="textSecondary">Type</Typography>
                        </TableCell>}
                        <TableCell>
                            <Typography color="textSecondary">Group</Typography>
                        </TableCell>
                    </TableRow>
                </TableHead>
                <TableBody>
                    {variables.map((v: any) => <InheritedVariableListItem
                        key={v.id}
                        fragmentRef={v}
                        showValues={showValues}
                    />)}
                </TableBody>
            </Table>
        </TableContainer>
    );
}

export default InheritedVariableList;