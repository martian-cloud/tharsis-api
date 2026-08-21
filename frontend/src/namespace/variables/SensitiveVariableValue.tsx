import { Box, CircularProgress, Link, Tooltip, Typography } from '@mui/material';
import LockIcon from '@mui/icons-material/LockOutlined';
import graphql from 'babel-plugin-relay/macro';
import { Suspense, useEffect, useState } from 'react';
import { ErrorBoundary } from 'react-error-boundary';
import { useLazyLoadQuery } from 'react-relay/hooks';
import CopyButton from '../../common/CopyButton';
import { SensitiveVariableValueQuery } from './__generated__/SensitiveVariableValueQuery.graphql';

interface Props {
    variableVersionId: string
}

function SensitiveVariableValue({ variableVersionId }: Props) {
    const [sensitiveValue, setSensitiveValue] = useState<{ id: string, value: string } | null>(null);
    const [loaded, setLoaded] = useState(false);
    const data = useLazyLoadQuery<SensitiveVariableValueQuery>(graphql`
        query SensitiveVariableValueQuery($id: String!, $includeSensitiveValue: Boolean!) {
            namespaceVariableVersion(id: $id, includeSensitiveValue: $includeSensitiveValue) {
                id
                value
            }
        }`, { id: variableVersionId, includeSensitiveValue: true }, { fetchPolicy: 'network-only' });

    useEffect(() => {
        // Set the sensitive value if it is not already set and the data has loaded
        if (data.namespaceVariableVersion?.value && (!sensitiveValue || data.namespaceVariableVersion.id !== sensitiveValue.id)) {
            // The sensitive value must be stored in state to prevent it from
            // being cleared out when another component loads the variable version
            // without the value field included.
            setSensitiveValue({
                id: data.namespaceVariableVersion.id,
                value: data.namespaceVariableVersion.value
            });
        }
        if (data) {
            setLoaded(true);
        }
    }, [data, sensitiveValue]);

    if (!loaded) {
        return null;
    } else if (sensitiveValue) {
        return (<>
            {sensitiveValue.value}
            <CopyButton
                data={sensitiveValue.value}
                toolTip="Copy value"
            />
        </>);
    } else {
        return <Typography variant="caption" color="error" noWrap>[variable has been deleted]</Typography>;
    }
}

function SensitiveVariableValueContainer({ variableVersionId }: Props) {
    const [showValue, setShowValue] = useState(false);
    return (
        <Box minWidth={100}>
            <ErrorBoundary fallbackRender={({ error }) => {
                if (!error?.codes?.includes('FORBIDDEN')) {
                    throw error;
                }
                return (
                    <Tooltip title="You do not have permission to view this sensitive value" placement="top">
                        <LockIcon color="disabled" />
                    </Tooltip>
                );
            }}>
                <Suspense fallback={<CircularProgress size={18} />}>
                    {!showValue && <Link
                        onClick={() => setShowValue(true)}
                        underline="hover"
                        color="secondary"
                        sx={{ cursor: 'pointer' }}>
                        View Secret
                    </Link>}
                    {showValue && <SensitiveVariableValue variableVersionId={variableVersionId} />}
                </Suspense>
            </ErrorBoundary>
        </Box>
    );
}

export default SensitiveVariableValueContainer;
