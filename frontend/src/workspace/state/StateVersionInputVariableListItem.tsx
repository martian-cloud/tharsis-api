import LockIcon from '@mui/icons-material/LockOutlined';
import { Box, Chip } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { MASKED_VALUE, monoFontFamily } from '../../common/DataTableCell';
import { ResponsiveRow } from '../../common/ResponsiveTable';
import SensitiveVariableValue from '../../namespace/variables/SensitiveVariableValue';
import Link from '../../routes/Link';
import { StateVersionInputVariableListItemFragment_variable$key } from './__generated__/StateVersionInputVariableListItemFragment_variable.graphql';

interface Props {
    fragmentRef: StateVersionInputVariableListItemFragment_variable$key;
    showValues: boolean;
}

function StateVersionInputVariableListItem(props: Props) {
    const { showValues } = props;
    const data = useFragment<StateVersionInputVariableListItemFragment_variable$key>(
        graphql`
        fragment StateVersionInputVariableListItemFragment_variable on RunVariable
        {
            key
            value
            category
            namespacePath
            sensitive
            versionId
            includedInTfConfig
        }
      `, props.fragmentRef);

    return (
        <ResponsiveRow cells={[
            {
                primary: true, content: <Box sx={{ wordBreak: 'break-word' }}>
                    {data.key}
                    {data.sensitive && <Chip sx={{ ml: 1 }} color="warning" size="xs" label="Sensitive" />}
                    {data.category === 'terraform' && data.includedInTfConfig === false && <Chip sx={{ ml: 1 }} color="warning" size="xs" label="Not used" />}
                </Box>
            },
            {
                label: 'Value', content: <Box sx={{ wordBreak: 'break-word', fontFamily: showValues ? monoFontFamily : undefined }}>
                    {!showValues && MASKED_VALUE}
                    {showValues && <>
                        {data.value === null && !data.sensitive && <LockIcon color="disabled" />}
                        {data.value !== null && !data.sensitive && data.value}
                        {data.sensitive && <SensitiveVariableValue variableVersionId={data.versionId as string} />}
                    </>}
                </Box>
            },
            {
                label: 'Source', content: <Box sx={{ wordBreak: 'break-word' }}>
                    {data.namespacePath && <Link
                        to={`/groups/${data.namespacePath}/-/variables`}
                        color="inherit"
                        variant="body1"
                    >
                        {data.namespacePath}
                    </Link>}
                    {!data.namespacePath && 'Run'}
                </Box>
            },
        ]} />
    );
}

export default StateVersionInputVariableListItem;
