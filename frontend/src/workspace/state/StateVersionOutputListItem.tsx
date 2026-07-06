import { Box } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { MASKED_VALUE, monoFontFamily } from '../../common/DataTableCell';
import { ResponsiveRow } from '../../common/ResponsiveTable';
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
    const masked = !showValues && data.sensitive;

    return (
        <ResponsiveRow cells={[
            { primary: true, content: <Box sx={{ wordBreak: 'break-word', fontFamily: monoFontFamily }}>{data.name}</Box> },
            { label: 'Value', content: <Box sx={{ wordBreak: 'break-word', fontFamily: masked ? undefined : monoFontFamily }}>{masked ? MASKED_VALUE : value}</Box> },
        ]} />
    );
}

export default StateVersionOutputListItem;
