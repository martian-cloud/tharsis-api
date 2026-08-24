import { Chip, SxProps, Theme } from '@mui/material';
import { alpha, styled } from '@mui/material/styles';
import { ElementType, ReactElement, ReactNode } from 'react';

// Pill sizes from the design: the default is the 24px status pill in a panel header, `small` the
// 20px inline pill used for enforcement levels, counts, and the OPA badge.
type PillSize = 'small' | 'medium';

// Visual treatments, named for the role rather than the colour so callers stay readable:
//  - solid: filled with the semantic colour, for a policy's own verdict
//  - tint:  a translucent wash of the semantic colour, for check status and progress labels
//  - outline: transparent with a neutral border, for metadata that shouldn't compete for attention
type PillVariant = 'solid' | 'tint' | 'outline';

interface Props {
    children: ReactNode;
    // A resolved colour (e.g. theme.palette.error.main). Ignored by the outline variant.
    color?: string;
    variant?: PillVariant;
    size?: PillSize;
    // An element rather than a ReactNode, because it goes in Chip's icon slot.
    icon?: ReactElement;
    sx?: SxProps<Theme>;
}

// RunStageStatusTypes reports colours as palette paths (e.g. 'runStatus.pending'), which sx accepts
// but alpha() does not. Resolve to a real colour before tinting. Mirrors RunStageIcons.
export function resolvePaletteColor(theme: Theme, path: string): string {
    return (path.split('.').reduce<any>((o, k) => (o == null ? o : o[k]), theme.palette) as string)
        ?? theme.palette.text.primary;
}

// Both sizes carry caption-sized text — the size prop varies the pill box, not the typography, which
// is how MuiChip's own xs variant in the theme is put together (height 20, 0.75rem).
const SIZES: Record<PillSize, { height: number; padding: string }> = {
    small: { height: 20, padding: '0 8px' },
    medium: { height: 24, padding: '0 10px' },
};

function treatment(variant: PillVariant, color: string | undefined, theme: Theme) {
    switch (variant) {
        case 'solid':
            return {
                background: color,
                // Filled pills carry dark text so they stay legible on the light-ish semantic
                // fills (success/warning) this app uses in dark mode.
                color: theme.palette.getContrastText(color ?? theme.palette.background.paper),
                border: '1px solid transparent',
            };
        case 'outline':
            return {
                background: 'transparent',
                color: theme.palette.text.secondary,
                border: `1px solid ${alpha(theme.palette.common.white, 0.23)}`,
                fontWeight: 400,
            };
        case 'tint':
        default:
            return {
                background: alpha(color ?? theme.palette.text.primary, 0.12),
                color,
                border: '1px solid transparent',
            };
    }
}

// The pill props are prefixed and kept off the DOM because Chip already owns `variant`, `size`, and
// `color` with meanings of its own — none of Chip's three are ever passed through. Chip's geometry is
// replaced outright rather than adjusted: its 32px height, 16px radius, 13px text, and label/icon
// padding all belong to a chip that can be clicked and deleted, which this never is.
const StyledChip = styled(Chip, {
    shouldForwardProp: prop => !['pillSize', 'pillVariant', 'pillColor'].includes(prop as string),
})<{
    pillSize: PillSize;
    pillVariant: PillVariant;
    pillColor?: string;
    // styled() flattens Chip's overridable-component typing, so the prop Chip still honours at
    // runtime has to be re-declared here to stay assignable.
    component?: ElementType;
}>(
    ({ theme, pillSize, pillVariant, pillColor }) => ({
        height: SIZES[pillSize].height,
        // The label's own padding is dropped below, so the box padding lives here and `gap` is what
        // separates an icon from its text.
        padding: SIZES[pillSize].padding,
        gap: '5px',
        borderRadius: '9999px',
        fontSize: theme.typography.caption.fontSize,
        fontWeight: 600,
        letterSpacing: '0.02em',
        lineHeight: 1,
        flexShrink: 0,
        '& .MuiChip-label': {
            padding: 0,
        },
        // Chip nudges its icon with a 5px/-6px margin pair to sit inside the label's padding; with
        // that padding gone the margins would double-count. The icon takes the pill's colour.
        '& .MuiChip-icon': {
            margin: 0,
            color: 'inherit',
        },
        ...treatment(pillVariant, pillColor, theme),
    })
);

// Pill is a rounded label: a status, a count, a verdict, or a type badge. It is deliberately inert —
// no click, no delete, no focus ring — so it stays legal inside a heading or a row that is itself a
// link, and it takes a resolved color rather than a palette name because the statuses it labels come
// from the app's own runStatus/checkResult palettes.
function Pill({ children, color, variant = 'tint', size = 'medium', icon, sx }: Props) {
    return (
        <StyledChip
            // A span rather than Chip's div, so a pill can sit inline beside text.
            component="span"
            label={children}
            icon={icon}
            pillSize={size}
            pillVariant={variant}
            pillColor={color}
            sx={sx}
        />
    );
}

export default Pill;
