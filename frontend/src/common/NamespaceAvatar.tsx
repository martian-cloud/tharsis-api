import { Avatar, Box, BoxProps } from '@mui/material';
import { darken, useTheme } from '@mui/material/styles';
import { forwardRef } from 'react';

export type NamespaceAvatarSize = 'xs' | 'sm' | 'md' | 'lg';

// 'default' is the flat avatar colour; 'gradient' is the primary gradient used in the nav drawers.
type NamespaceAvatarAppearance = 'default' | 'gradient';

// box is the footprint for both types, so groups and workspaces line up. The avatar is `offset`
// smaller than the box for both types, leaving room for the tile stacked behind a group's avatar.
const SIZES: Record<NamespaceAvatarSize, { box: number; offset: number; border: number; twoLetterFontSize: number }> = {
    xs: { box: 24, offset: 2, border: 1, twoLetterFontSize: 11 },
    sm: { box: 28, offset: 2, border: 1, twoLetterFontSize: 12 },
    md: { box: 32, offset: 3, border: 1, twoLetterFontSize: 14 },
    lg: { box: 56, offset: 6, border: 2, twoLetterFontSize: 22 },
};

// getNamespaceInitials returns the first letter of the first two '-' or '_' separated segments of
// a name (prod-network => PN), or just the first letter when the name has a single segment.
export function getNamespaceInitials(name: string): string {
    return name
        .split(/[-_]/)
        .filter(segment => segment.length > 0)
        .slice(0, 2)
        .map(segment => segment[0].toUpperCase())
        .join('');
}

interface Props extends Omit<BoxProps, 'children'> {
    name: string;
    // Groups are drawn as a stack of namespaces; workspaces as a single avatar
    type: 'group' | 'workspace';
    size?: NamespaceAvatarSize;
    appearance?: NamespaceAvatarAppearance;
}

const NamespaceAvatar = forwardRef<HTMLDivElement, Props>(function NamespaceAvatar(
    { name, type, size = 'md', appearance = 'default', sx, ...boxProps },
    ref
) {
    const theme = useTheme();
    const { box, offset, border, twoLetterFontSize } = SIZES[size];
    const isGroup = type === 'group';
    const avatarSize = box - offset;
    const initials = getNamespaceInitials(name);
    const gradient = appearance === 'gradient';

    return (
        <Box
            ref={ref}
            {...boxProps}
            sx={[
                { position: 'relative', width: box, height: box, flexShrink: 0 },
                ...(Array.isArray(sx) ? sx : [sx])
            ]}
        >
            {isGroup && (
                <Box sx={{
                    position: 'absolute',
                    left: offset,
                    top: 0,
                    width: avatarSize,
                    height: avatarSize,
                    borderRadius: gradient ? '6px' : 1,
                    bgcolor: darken(gradient ? theme.palette.primary.dark : theme.palette.avatar.default, gradient ? 0.4 : 0.5)
                }} />
            )}
            <Avatar
                variant="rounded"
                sx={{
                    position: 'absolute',
                    left: 0,
                    top: offset,
                    width: avatarSize,
                    height: avatarSize,
                    ...(initials.length > 1 && { fontSize: twoLetterFontSize, fontWeight: 600, letterSpacing: '-0.5px' }),
                    ...(gradient ? {
                        borderRadius: '6px',
                        background: `linear-gradient(135deg, ${theme.palette.primary.dark}, ${theme.palette.primary.main})`,
                        ...(initials.length === 1 && { fontSize: theme.typography.caption.fontSize }),
                        fontWeight: 600,
                        color: theme.palette.primary.contrastText,
                    } : { bgcolor: 'avatar.default' }),
                    ...(isGroup && {
                        boxSizing: 'border-box',
                        border: `${border}px solid ${theme.palette.background.default}`
                    })
                }}
            >
                {initials}
            </Avatar>
        </Box>
    );
});

export default NamespaceAvatar;
