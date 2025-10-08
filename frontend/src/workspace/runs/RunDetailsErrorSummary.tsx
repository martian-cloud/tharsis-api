import { Paper, useTheme } from '@mui/material';
import { darken } from '@mui/material/styles';
import LogViewer from './LogViewer';

interface Props {
    ml?: number
    mr?: number
    mb?: number
    errorMessage: string
}

function RunDetailsErrorSummary({ errorMessage, ml, mr, mb }: Props) {
    const theme = useTheme();
    return (
        <Paper sx={{
            backgroundColor: darken(theme.palette.background.default, 0.3),
            ml,
            mr,
            mb,
            pt: 1,
            pr: 2,
            pb: 1,
            pl: 2
        }}>
            <LogViewer
                logs={errorMessage}
                hideLineNumbers
            />
        </Paper>
    );
}

export default RunDetailsErrorSummary;
