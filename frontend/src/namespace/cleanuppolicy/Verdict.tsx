import { CleanupRule } from './types';
import { Value } from './ConditionText';

interface Props {
    rule: CleanupRule;
    resource: string;
}

function Verdict({ rule, resource }: Props) {
    const keepMin = (rule as { keepMin?: number }).keepMin ?? 0;
    const noun = keepMin === 1 ? resource : `${resource}s`;

    switch (rule.strategy) {
        case 'PROTECT':
            return <>never delete {noun}</>;
        case 'COUNT':
            return <>keep newest <Value>{keepMin}</Value> {noun}</>;
        case 'AGE':
            return keepMin > 0
                ? <>delete {noun} after <Value>{rule.deleteAfterDays}d</Value>, keep newest <Value>{keepMin}</Value></>
                : <>delete {noun} after <Value>{rule.deleteAfterDays}d</Value></>;
        default:
            return <>unrecognized strategy, no {resource} is deleted</>;
    }
}

export default Verdict;
