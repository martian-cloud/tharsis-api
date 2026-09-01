/**
 * ConditionText.tsx — renders one rule condition as prose (e.g. "name matches aws-*"), muting the
 * label and highlighting the literal value so it reads as a value rather than blending into the
 * surrounding phrase.
 */
import { Typography } from '@mui/material';
import { Condition } from './types';

// Value highlights one literal piece of text in a rule's description -- a number, a duration, a
// glob, or a status -- so it reads as a literal rather than blending into the surrounding prose.
// Exported so Verdict can highlight its own numbers the same way.
export function Value({ children }: { children: React.ReactNode }) {
    return <Typography component="span" variant="code" color="primary.main">{children}</Typography>;
}

// ConditionText renders one condition as prose (a muted label, and — if the condition carries a
// literal value like a glob or a status — that value highlighted via Value), shared by the rule
// list's ConditionBadge and the dialog's summary sentence which differ only in how they wrap it.
function ConditionText({ label, value }: Condition) {
    return (
        <>
            <Typography component="span" variant="caption" color="text.secondary">{label}</Typography>
            {value && <>{' '}<Value>{value}</Value></>}
        </>
    );
}

export default ConditionText;
