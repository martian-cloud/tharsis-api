import { useState } from 'react';
import { IconButton, Tooltip } from '@mui/material';
import CopyIcon from '@mui/icons-material/ContentCopy';
import { SxProps } from '@mui/system';

interface Props {
    data: string
    toolTip: string
    sxCopyIconStyles?: SxProps
}

function CopyButton({ data, toolTip, sxCopyIconStyles = { width: 16, height: 16 } }: Props) {
    const [showCopyText, setShowCopyText] = useState(false);
    const [tooltipTitle, setTooltipTitle] = useState(toolTip);

    const handleClickCopyIcon = () => {
        navigator.clipboard.writeText(data);
        setTooltipTitle("Copied");
        setShowCopyText(true);
        setTimeout(() => {
            setShowCopyText(false);
            setTooltipTitle(toolTip);
        }, 2000);
    };

    const handleMouseEnter = () => {
        setTooltipTitle(toolTip);
        setShowCopyText(true);
    };

    const handleMouseLeave = () => {
        setShowCopyText(false);
    };

    return (
        <Tooltip
            title={tooltipTitle}
            placement="top"
            open={showCopyText}
            enterDelay={0}
            leaveDelay={0}
        >
            <IconButton
                sx={{
                    opacity: '20%',
                    transition: 'ease',
                    transitionDuration: '300ms',
                    ":hover": {
                        opacity: '100%'
                    }
                }}
                onClick={handleClickCopyIcon}
                onMouseEnter={handleMouseEnter}
                onMouseLeave={handleMouseLeave}
            >
                <CopyIcon sx={{ ...sxCopyIconStyles }} />
            </IconButton>
        </Tooltip>
    );
}

export default CopyButton;
