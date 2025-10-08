import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Typography from '@mui/material/Typography';
import VariableListItem from './VariableListItem';

interface Props {
    variables: any[]
    namespacePath: string
    showValues: boolean
    onShowHistory: (variable: any) => void;
    onEditVariable: (variable: any) => void
    onDeleteVariable: (variable: any) => void
}

function VariableList(props: Props) {
    const { variables, namespacePath, showValues, onEditVariable, onDeleteVariable, onShowHistory } = props;

    return (
        <TableContainer>
            <Table>
                <TableHead>
                    <TableRow>
                        <TableCell>
                            <Typography color="textSecondary">Key</Typography>
                        </TableCell>
                        <TableCell>
                            <Typography color="textSecondary">Value</Typography>
                        </TableCell>
                        <TableCell>
                            <Typography color="textSecondary">Source</Typography>
                        </TableCell>
                        <TableCell></TableCell>
                    </TableRow>
                </TableHead>
                <TableBody>
                    {variables.map((v: any) => <VariableListItem
                        key={v.id}
                        fragmentRef={v}
                        namespacePath={namespacePath}
                        showValues={showValues}
                        onEdit={onEditVariable}
                        onDelete={onDeleteVariable}
                        onShowHistory={onShowHistory}
                    />)}
                </TableBody>
            </Table>
        </TableContainer>
    );
}

export default VariableList;
