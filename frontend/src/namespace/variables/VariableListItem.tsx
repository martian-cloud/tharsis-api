import DeleteIcon from '@mui/icons-material/CloseOutlined';
import EditIcon from '@mui/icons-material/EditOutlined';
import HistoryIcon from '@mui/icons-material/HistoryOutlined';
import LockIcon from '@mui/icons-material/LockOutlined';
import { Box, Button, Chip, Stack, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import CopyButton from '../../common/CopyButton';
import { MASKED_VALUE, monoFontFamily } from '../../common/DataTableCell';
import { ResponsiveRow } from '../../common/ResponsiveTable';
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

    const keyContent = (
        <Box>
            <Box display="flex" alignItems="center" sx={{ fontWeight: 'bold', fontFamily: monoFontFamily, wordBreak: 'break-all' }}>
                {data.key}
                <CopyButton
                    data={data.key}
                    toolTip="Copy key"
                />
            </Box>
            {data.sensitive && <Chip color="warning" sx={{ mt: 0.5, fontWeight: 'bold' }} size="xs" label="Sensitive" />}
        </Box>
    );

    const valueContent = (
        <Box sx={{ fontFamily: monoFontFamily, wordBreak: 'break-all' }}>
            {!showValues && MASKED_VALUE}
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
    );

    const source = data.namespacePath === namespacePath ? <Typography variant="body2" color="textSecondary">Direct</Typography> : <Link
        to={`/groups/${data.namespacePath}/-/variables`}
        color="inherit"
        variant="body1"
    >
        {data.namespacePath}
    </Link>;

    const actions = data.namespacePath === namespacePath ? <Stack direction="row" spacing={1} justifyContent="flex-end">
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
    </Stack> : null;

    return (
        <ResponsiveRow cells={[
            { primary: true, content: keyContent },
            { label: 'Value', content: valueContent },
            { label: 'Source', content: source },
            { align: 'right', content: actions },
        ]} />
    );
}

export default VariableListItem;
