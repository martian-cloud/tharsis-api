import Typography from '@mui/material/Typography';
import Box from '@mui/material/Box';
import React, { ReactNode } from 'react';
import ComplexityLimit from './ComplexityLimit'
import MaintenanceIcon from '@mui/icons-material/Engineering';

const COMPLEXITY_EXCEEDED_ERROR = 'RATE_LIMIT_EXCEEDED';
const NOT_IMPLEMENTED_ERROR = 'NOT_IMPLEMENTED';

interface Props {
    children?: ReactNode;
}

interface State {
    hasError: boolean
    errorCodes: string[]
}

class ErrorBoundary extends React.Component<Props, State> {
    constructor(props: any) {
        super(props);
        this.state = { hasError: false, errorCodes: [] };
    }

    static getDerivedStateFromError(error: any) {
        // Update state so the next render will show the fallback UI.
        return { hasError: true, errorCodes: (error && error.codes) ? error.codes : [] };
    }

    componentDidCatch(error: any, errorInfo: any) {
        console.log(`Unexpected error: ${error}: ${errorInfo}`);
    }

    render() {
        if (this.state.hasError) {
            if (this.state.errorCodes.includes(NOT_IMPLEMENTED_ERROR)) {
                return (<Box padding={4} display="flex" justifyContent="center" alignItems="center" height="400px" flexDirection="column">
                    <MaintenanceIcon fontSize="large" sx={{ mb: 2 }} />
                    <Typography align="center" variant="h6" maxWidth={800} color="textSecondary">
                        Tharsis is attempting to access a feature that doesn't exist in the API. This most likely indicates a maintenance update is in progress,
                        please check back shortly.
                    </Typography>
                </Box>);
            } else if (this.state.errorCodes.includes(COMPLEXITY_EXCEEDED_ERROR)) {
                return <ComplexityLimit />;
            } else if (this.state.errorCodes.includes("NOT_FOUND")) {
                return (<Box padding={4} display="flex" justifyContent="center" alignItems="center" height="400px">
                    <Typography variant="h6">
                        The resource you're attempting to view either doesn't exist or you don't have access to view it.
                    </Typography>
                </Box>);
            } else {
                return (<Box padding={4} display="flex" justifyContent="center" alignItems="center" height="400px">
                    <Typography variant="h5">
                        Oops! Something went wrong. Please reload the page and try again.
                    </Typography>
                </Box>);
            }
        }
        return this.props.children;
    }

}

export default ErrorBoundary
