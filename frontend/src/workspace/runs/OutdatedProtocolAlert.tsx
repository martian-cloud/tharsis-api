import { Alert } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { OutdatedProtocolAlertFragment_job$key } from './__generated__/OutdatedProtocolAlertFragment_job.graphql';

interface Props {
    fragmentRef: OutdatedProtocolAlertFragment_job$key;
}

function OutdatedProtocolAlert({ fragmentRef }: Props) {
    const data = useFragment<OutdatedProtocolAlertFragment_job$key>(
        graphql`
            fragment OutdatedProtocolAlertFragment_job on Job {
                outdatedJobProtocolVersion
            }
        `,
        fragmentRef
    );

    if (!data.outdatedJobProtocolVersion) {
        return null;
    }

    return (
        <Alert severity="warning" variant="outlined" sx={{ mb: 2 }}>
            This job was executed using an older version of the job executor which does not have all the latest capabilities.
        </Alert>
    );
}

export default OutdatedProtocolAlert;
