import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import graphql from 'babel-plugin-relay/macro';
import React, { useMemo } from 'react';
import { useFragment } from 'react-relay/hooks';
import { getCheckStatusTooltip, collectFailureMessages } from '../checks/checks';
import CheckStatusChip from '../checks/CheckStatusChip';
import FailureMessagesList from '../checks/FailureMessagesList';
import { ResponsiveRow } from '../../common/ResponsiveTable';
import { StateVersionCheckResultRowFragment_checkResult$key } from './__generated__/StateVersionCheckResultRowFragment_checkResult.graphql';

interface Props {
    fragmentRef: StateVersionCheckResultRowFragment_checkResult$key
}

function StateVersionCheckResultRow(props: Props) {
    const { fragmentRef } = props;
    const data = useFragment<StateVersionCheckResultRowFragment_checkResult$key>(
        graphql`
        fragment StateVersionCheckResultRowFragment_checkResult on CheckResult
        {
            name
            status
            objects {
                address
                status
                failureMessages
            }
        }
      `, fragmentRef);

    const failureMessages = useMemo(() => {
        return collectFailureMessages(data.objects);
    }, [data.objects]);

    return (
        <ResponsiveRow cells={[
            {
                label: 'Status',
                content: (
                    <Tooltip title={getCheckStatusTooltip(data.status)} placement="top">
                        <span>
                            <CheckStatusChip status={data.status} />
                        </span>
                    </Tooltip>
                ),
            },
            {
                label: 'Name',
                primary: true,
                content: (
                    <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                        {data.name}
                    </Typography>
                ),
            },
            {
                label: 'Details',
                content: failureMessages.length > 0 ? (
                    <FailureMessagesList messages={failureMessages} />
                ) : null,
            },
        ]} />
    );
}

export default StateVersionCheckResultRow;
