import Typography from '@mui/material/Typography';
import Box from '@mui/material/Box';
import React, { ReactNode } from 'react';
import ComplexityLimit from './ComplexityLimit'

const COMPLEXITY_EXCEEDED_ERROR = 'RATE_LIMIT_EXCEEDED';

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
            return this.state.errorCodes.includes(COMPLEXITY_EXCEEDED_ERROR) ?
                <ComplexityLimit /> :
                (<Box padding={4} display="flex" justifyContent="center" alignItems="center" height="400px">
                    <Typography variant="h5">Oops! Something went wrong. Please reload the page and try again.</Typography>
                </Box>);
        }
        return this.props.children;
    }
}

export default ErrorBoundary
