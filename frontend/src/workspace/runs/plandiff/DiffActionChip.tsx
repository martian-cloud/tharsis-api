import { Chip } from '@mui/material';
import colors from './RunDetailsPlanDiffColors';
import { PlanChangeAction } from './__generated__/RunDetailsPlanDiffViewerFragment_run.graphql';

interface Props {
    action: PlanChangeAction
    importing: boolean
}

function DiffActionChip({ action, importing }: Props) {
    let label = 'unknown';

    let color = 'inherit';
    switch (action) {
        case 'CREATE':
            label = 'create';
            color = colors.create;
            break;
        case 'CREATE_THEN_DELETE':
            label = 'create then delete';
            color = colors.delete;
            break;
        case 'DELETE':
            label = 'delete';
            color = colors.delete;
            break;
        case 'DELETE_THEN_CREATE':
            label = 'delete then create';
            color = colors.delete;
            break;
        case 'NOOP':
            label = importing ? 'import' : 'no changes';
            color = importing ? colors.import : colors.noop;
            break;
        case 'READ':
            label = 'read';
            color = colors.read;
            break;
        case 'UPDATE':
            label = importing ? 'import and update' : 'update';
            color = colors.update;
            break;
    }

    return (
        <Chip
            label={label}
            variant="outlined"
            size="xs"
            sx={{
                color: color,
                borderColor: color
            }}
        />
    );
}

export default DiffActionChip;
