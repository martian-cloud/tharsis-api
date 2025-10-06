import DeleteIcon from '@mui/icons-material/CloseOutlined';
import EditIcon from '@mui/icons-material/EditOutlined';
import HistoryIcon from '@mui/icons-material/HistoryOutlined';
import LockIcon from '@mui/icons-material/LockOutlined';
import { Box, Chip, Stack, Typography } from '@mui/material';
import Button from '@mui/material/Button';
import TableCell from '@mui/material/TableCell';
import TableRow from '@mui/material/TableRow';
import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import CopyButton from '../../common/CopyButton';
import DataTableCell from '../../common/DataTableCell';
import Link from '../../routes/Link';
import { VariableListItemFragment_variable$key } from './__generated__/VariableListItemFragment_variable.graphql';
import SensitiveVariableValue from './SensitiveVariableValue';

interface Props {
    fragmentRef: VariableListItemFragment_variable$key;
    namespacePath: string;
    showValues: boolean;
    onShowHistory: (variable: any) => void;
    onEdit: (variable: any) => void;
    onDelete: (variable: any) => void;
}

function VariableListItem(props: Props) {
    const { onEdit, onDelete, onShowHistory, namespacePath, showValues } = props;
    const data = useFragment<VariableListItemFragment_variable$key>(
        graphql`
        fragment VariableListItemFragment_variable on NamespaceVariable
        {
            id
            key
            category
            sensitive
            value
            namespacePath
            latestVersionId
            metadata {
                updatedAt
            }
        }
      `, props.fragmentRef);

    return (
        <TableRow
            sx={{ '&:last-child td, &:last-child th': { border: 0 }, height: 64 }}
        >
            <DataTableCell>
                <Box minWidth={200} display="flex" alignItems="center" sx={{ fontWeight: 'bold', wordBreak: 'break-all' }}>
                    {data.key}
                    <CopyButton
                        data={data.key}
                        toolTip="Copy key"
                    />
                </Box>
                {data.sensitive && <Chip color="warning" sx={{ mt: 0.5, fontWeight: 'bold' }} size="xs" label="Sensitive" />}
            </DataTableCell>
            <DataTableCell sx={{ wordBreak: 'break-all' }}>
                <Box minWidth={200}>
                    {!showValues && '********'}
                    {showValues && <>
                        {data.value === null && !data.sensitive && <LockIcon color="disabled" />}
                        {data.value !== null && !data.sensitive && <React.Fragment>
                            {data.value}
                        </React.Fragment>}
                        {data.sensitive && <SensitiveVariableValue variableVersionId={data.latestVersionId} />}
                    </>}
                    {data.value != null && <CopyButton
                        data={data.value}
                        toolTip="Copy value"
                    />}
                </Box>
            </DataTableCell>
            <TableCell>
                {data.namespacePath === namespacePath ? <Typography variant="body2" color="textSecondary">Direct</Typography> : <Link
                    to={`/groups/${data.namespacePath}/-/variables`}
                    color="inherit"
                    variant="body1"
                >
                    {data.namespacePath}
                </Link>}
            </TableCell>
            <TableCell align='right'>
                {data.namespacePath === namespacePath && <Stack direction="row" spacing={1} justifyContent="flex-end">
                    <Button
                        onClick={() => onEdit(data)}
                        sx={{ minWidth: 40, padding: '2px' }}
                        size="small"
                        color="info"
                        variant="outlined">
                        <EditIcon />
                    </Button>
                    <Button
                        onClick={() => onShowHistory(data)}
                        sx={{ minWidth: 40, padding: '2px' }}
                        size="small"
                        color="info"
                        variant="outlined">
                        <HistoryIcon />
                    </Button>
                    <Button
                        onClick={() => onDelete(data)}
                        sx={{ minWidth: 40, padding: '2px' }}
                        size="small"
                        color="info"
                        variant="outlined">
                        <DeleteIcon />
                    </Button>
                </Stack>}
            </TableCell>
        </TableRow>
    );
}

export default VariableListItem;
