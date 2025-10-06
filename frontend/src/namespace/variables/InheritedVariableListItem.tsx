import LockIcon from '@mui/icons-material/LockOutlined';
import TableCell from '@mui/material/TableCell';
import TableRow from '@mui/material/TableRow';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import DataTableCell from '../../common/DataTableCell';
import Link from '../../routes/Link';
import { InheritedVariableListItemFragment_variable$key } from './__generated__/InheritedVariableListItemFragment_variable.graphql';

interface Props {
    fragmentRef: InheritedVariableListItemFragment_variable$key;
    showValues: boolean;
}

function InheritedVariableListItem(props: Props) {
    const { showValues } = props;
    const data = useFragment<InheritedVariableListItemFragment_variable$key>(
        graphql`
        fragment InheritedVariableListItemFragment_variable on NamespaceVariable
        {
            id
            key
            category
            value
            namespacePath
        }
      `, props.fragmentRef);

    return (
        <TableRow
            sx={{ '&:last-child td, &:last-child th': { border: 0 }, height: 64 }}
        >
            <DataTableCell sx={{ fontWeight: 'bold', wordBreak: 'break-all' }}>
                {data.key}
            </DataTableCell>
            <DataTableCell sx={{ wordBreak: 'break-all' }} mask={!showValues} >
                {data.value !== null ? data.value : <LockIcon color="disabled" />}
            </DataTableCell>
            <TableCell>
                <Link
                    to={`/groups/${data.namespacePath}/-/variables`}
                    color="inherit"
                    variant="body1"
                >
                    {data.namespacePath}
                </Link>
            </TableCell>
        </TableRow>
    );
}

export default InheritedVariableListItem;
