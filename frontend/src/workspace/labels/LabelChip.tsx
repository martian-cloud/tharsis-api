import { Chip, Tooltip } from '@mui/material';
import { useTheme } from '@mui/material/styles';

interface Props {
    labelKey: string;
    value: string;
    size?: 'small' | 'medium' | 'xs';
    variant?: 'filled' | 'outlined';
    maxValueLength?: number;
    showKeyOnly?: boolean;
}

function LabelChip({ labelKey, value, size = 'small', variant = 'outlined', maxValueLength = 20, showKeyOnly = false }: Props) {
    const theme = useTheme();

    // Determine display text based on showKeyOnly
    const displayText = showKeyOnly ? labelKey : `${labelKey}: ${value.length > maxValueLength ? `${value.substring(0, maxValueLength)}...` : value}`;
    const tooltipText = `${labelKey}: ${value}`;

    return (
        <Tooltip title={tooltipText} arrow>
            <Chip
                size={size}
                variant={variant}
                label={displayText}
                sx={{
                    fontSize: size === 'xs' ? '0.75rem' : undefined,
                    height: size === 'xs' ? '20px' : undefined,
                    borderRadius: '4px',
                    backgroundColor: variant === 'filled' ? theme.palette.primary.main : 'transparent',
                    color: variant === 'filled' ? theme.palette.primary.contrastText : theme.palette.text.primary,
                    borderColor: theme.palette.divider,
                    '& .MuiChip-label': {
                        paddingLeft: size === 'xs' ? '6px' : undefined,
                        paddingRight: size === 'xs' ? '6px' : undefined,
                        fontWeight: 500,
                    }
                }}
            />
        </Tooltip>
    );
}

export default LabelChip;
