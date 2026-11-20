import { Box, Typography, useTheme } from '@mui/material';
import { alpha } from '@mui/material/styles';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { atomDark as prismTheme } from 'react-syntax-highlighter/dist/esm/styles/prism';
import CopyButton from '../../common/CopyButton';

interface Props {
    message: string
}

function VCSEventErrorMessage({ message }: Props) {
    const theme = useTheme();

    return (
        <Box sx={{ borderLeft: `3px solid ${theme.palette.error.main}`, background: alpha(theme.palette.error.main, 0.06), p: 2 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <Typography variant="caption" color="error">Error message</Typography>
                <CopyButton data={message} toolTip="Copy error message" />
            </Box>
            {/* width 0 + minWidth 100% keeps an unbreakable token from widening the table; the block scrolls instead. */}
            <Box sx={{ mt: 1, width: 0, minWidth: '100%' }}>
                <SyntaxHighlighter
                    language="text"
                    style={prismTheme}
                    wrapLongLines
                    tabIndex={0}
                    customStyle={{ margin: 0, borderRadius: 4, fontSize: 13, maxHeight: 320, overflow: 'auto' }}
                >
                    {message}
                </SyntaxHighlighter>
            </Box>
        </Box>
    );
}

export default VCSEventErrorMessage;
