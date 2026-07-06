import { ResponsiveTable } from '../../common/ResponsiveTable';
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
        <ResponsiveTable
            ariaLabel="variables"
            minWidth={650}
            columns={[{ label: 'Key' }, { label: 'Value' }, { label: 'Source' }, { label: '' }]}
        >
            {variables.map((v: any) => <VariableListItem
                key={v.id}
                fragmentRef={v}
                namespacePath={namespacePath}
                showValues={showValues}
                onEdit={onEditVariable}
                onDelete={onDeleteVariable}
                onShowHistory={onShowHistory}
            />)}
        </ResponsiveTable>
    );
}

export default VariableList;
