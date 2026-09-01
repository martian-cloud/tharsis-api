import { Fragment } from 'react';
import { Box, Tooltip, Typography } from '@mui/material';
import { CLEANUP_POLICY_KINDS } from './rules';
import { CleanupPolicyKind } from './types';
import PanelButton from '../../common/PanelButton';

interface Props {
    kinds: readonly CleanupPolicyKind[];
    selectedKind: CleanupPolicyKind | '';
    // Omit onSelect to render a fully read-only grid that just highlights the current kind
    // (used on the edit page, where the kind can't be changed after creation).
    onSelect?: (kind: CleanupPolicyKind) => void;
    // Kinds that should render but not be clickable, each mapped to a short explanation shown in a
    // tooltip (e.g. already has a local policy, or already inherited). Used on the new page so a
    // kind that isn't available right now is still visible, with a reason, rather than vanishing.
    disabledKinds?: ReadonlyMap<CleanupPolicyKind, string>;
}

// Renders the cleanup policy kinds as a grid of PanelButtons. The section header (e.g. "Type") is
// supplied by the caller via CleanupPolicyFormSection, so this component is just the grid itself.
function CleanupPolicyKindSelector({ kinds, selectedKind, onSelect, disabledKinds }: Props) {
    return (
        <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(220px, 1fr))', gap: 1.5 }}>
            {kinds.map(k => {
                const kindDef = CLEANUP_POLICY_KINDS[k];
                const selected = selectedKind === k;
                const disabledReason = disabledKinds?.get(k);
                const clickable = !!onSelect && !disabledReason;

                const card = (
                    <PanelButton
                        selected={selected}
                        disabled={!clickable}
                        onClick={clickable ? () => onSelect!(k) : undefined}
                        style={{
                            width: '100%',
                            maxWidth: 'none',
                            textAlign: 'center',
                            // A disabled kind can still be the selected one (e.g. arriving via an
                            // "Override Policy" link with the kind preselected) — in that case it
                            // should read as selected, not faded, even though it's not clickable.
                            opacity: disabledReason && !selected ? 0.5 : 1,
                        }}
                    >
                        <Typography variant="body2" fontWeight={600}>{kindDef.label}</Typography>
                        <Typography variant="caption" color="textSecondary">{kindDef.description}</Typography>
                    </PanelButton>
                );

                return (
                    <Fragment key={k}>
                        {disabledReason ? <Tooltip title={disabledReason}>{card}</Tooltip> : card}
                    </Fragment>
                );
            })}
        </Box>
    );
}

export default CleanupPolicyKindSelector;
