import { styled } from "@mui/material";
import { darken } from '@mui/material/styles';

interface PanelButtonProps {
    selected?: boolean
    disabled?: boolean
}

export const PanelButton = styled(
    'div',
    { shouldForwardProp: (prop) => !['selected', 'disabled'].includes(prop.toString()) }
)<PanelButtonProps>(({ theme, selected, disabled }) => ({
    border: `1px solid ${theme.palette.divider}`,
    borderRadius: 4,
    padding: 16,
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    maxWidth: 300,
    ...(selected && {
        borderColor: theme.palette.primary.main,
        backgroundColor: darken(theme.palette.primary.main, 0.8)
    }),
    ...(!disabled && {
        cursor: 'pointer',
        '&:hover': {
            backgroundColor: darken(theme.palette.primary.main, 0.8)
        },
    })
}));

export default PanelButton
