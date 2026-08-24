import { Box, Tooltip } from '@mui/material';
import { useTheme } from '@mui/material/styles';
import Pill from '../common/Pill';

// KIND_PILLS spells a package kind the short way the type pill does elsewhere in the app, keeping the
// long form for the tooltip. An unknown kind falls back to the raw enum so a type the api gained before
// the ui knew about it still shows something.
const KIND_PILLS: Record<string, { label: string, title: string }> = {
    OPA_POLICY: { label: 'OPA', title: 'Open Policy Agent (Rego) policy bundle' },
};

interface Props {
    // The PackageKind enum as it comes off the wire.
    kind: string
}

// PackageKindPill is the type badge that sits beside a package name. It is the same pill a run's policy
// card shows for the package behind the policy, so a package reads the same way wherever it appears.
function PackageKindPill({ kind }: Props) {
    const theme = useTheme();
    const pill = KIND_PILLS[kind] ?? { label: kind, title: 'Package type' };

    return (
        // Tooltip needs a ref and Pill is a plain function component, hence the span.
        <Tooltip title={pill.title}>
            <Box component="span" sx={{ display: 'inline-flex' }}>
                <Pill size="small" color={theme.palette.primary.main}>
                    {pill.label}
                </Pill>
            </Box>
        </Tooltip>
    );
}

export default PackageKindPill;
