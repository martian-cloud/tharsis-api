import MuiDrawer, { DrawerProps as MuiDrawerProps } from '@mui/material/Drawer';
import { styled } from '@mui/material/styles';
import { useAgentCopilot } from '../ai/AgentCopilotProvider';
import { useAppHeaderHeight } from '../contexts/AppHeaderHeightProvider';

interface DrawerProps extends MuiDrawerProps {
    width?: number | string;
    mobileWidth?: number | string;
    temporary?: boolean;
    hideWhenCopilotOpen?: boolean;
}

const StyledDrawer = styled(MuiDrawer, {
    shouldForwardProp: (prop) => !['width', 'mobileWidth', 'temporary'].includes(prop as string),
})<DrawerProps>(({ theme, width = 400, mobileWidth = 0, temporary = false }) => ({
    flexShrink: 0,
    overflowX: 'hidden',
    // Temporary drawers use full width, permanent drawers are responsive
    width: temporary ? width : mobileWidth,
    [`& .MuiDrawer-paper`]: {
        overflowX: 'hidden',
        boxSizing: 'border-box',
        width: temporary ? width : mobileWidth,
    },
    // Only permanent drawers get responsive behavior
    ...(!temporary && {
        [theme.breakpoints.up('md')]: {
            width: width,
            [`& .MuiDrawer-paper`]: {
                width: width,
            },
        }
    })
}));

function Drawer({
    children,
    width = 400,
    mobileWidth = 0,
    temporary = false,
    open = true,
    hideWhenCopilotOpen = false,
    variant,
    sx,
    ...otherProps
}: DrawerProps) {
    const { headerHeight } = useAppHeaderHeight();
    const { sidebarWidth, expanded } = useAgentCopilot();

    const isRight = otherProps.anchor === 'right';
    const hide = hideWhenCopilotOpen && expanded;

    // Header positioning + right offset for AI sidebar
    const headerStyles = {
        // When hiding a right drawer, collapse its width
        ...(hide && { width: 0 }),
        [`& .MuiDrawer-paper`]: {
            top: `${headerHeight}px`,
            height: `calc(100vh - ${headerHeight}px)`,
            ...(isRight && { right: `${sidebarWidth}px`, transition: 'right 0.2s ease, width 0.2s ease' }),
            ...(hide && { width: 0, overflow: 'hidden' }),
        },
    };

    // Merge with user sx
    const mergedSx = sx ? { ...headerStyles, ...sx } : headerStyles;

    return (
        <StyledDrawer
            width={hide ? 0 : width}
            mobileWidth={hide ? 0 : mobileWidth}
            temporary={temporary}
            variant={variant || (temporary ? 'temporary' : 'permanent')}
            open={hide ? false : open}
            sx={mergedSx}
            {...otherProps}
        >
            {children}
        </StyledDrawer>
    );
}

export default Drawer;
