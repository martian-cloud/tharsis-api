import { styled } from '@mui/material/styles';
import MuiDrawer, { DrawerProps as MuiDrawerProps } from '@mui/material/Drawer';
import { useAppHeaderHeight } from '../contexts/AppHeaderHeightProvider';

interface DrawerProps extends MuiDrawerProps {
    width?: number | string;
    mobileWidth?: number | string;
    temporary?: boolean;
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
    variant,
    sx,
    ...otherProps
}: DrawerProps) {
    const { headerHeight } = useAppHeaderHeight();

    // Header positioning
    const headerStyles = {
        [`& .MuiDrawer-paper`]: {
            top: `${headerHeight}px`,
            height: `calc(100vh - ${headerHeight}px)`,
        },
    };

    // Merge with user sx
    const mergedSx = sx ? { ...headerStyles, ...sx } : headerStyles;

    return (
        <StyledDrawer
            width={width}
            mobileWidth={mobileWidth}
            temporary={temporary}
            variant={variant || (temporary ? 'temporary' : 'permanent')}
            open={open}
            sx={mergedSx}
            {...otherProps}
        >
            {children}
        </StyledDrawer>
    );
}

export default Drawer;
