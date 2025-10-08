import React from 'react'
import { Alert, Stack, Typography } from '@mui/material'
import moment from 'moment';
import ForceCancelRunButton from './ForceCancelRunButton';
import { useFragment } from 'react-relay';
import graphql from 'babel-plugin-relay/macro';
import { ForceCancelRunAlertFragment_run$key } from './__generated__/ForceCancelRunAlertFragment_run.graphql'

interface Props {
    fragmentRef: ForceCancelRunAlertFragment_run$key
}

function ForceCancelRunAlert(props: Props) {

    const data = useFragment<ForceCancelRunAlertFragment_run$key>(
        graphql`
        fragment ForceCancelRunAlertFragment_run on Run
        {
            forceCancelAvailableAt
            ...ForceCancelRunButtonFragment_run
        }
        `, props.fragmentRef
    );

    const forceCancelAvailable = moment(data.forceCancelAvailableAt as moment.MomentInput).isSameOrBefore();

    return (
        <Alert severity='warning' sx={{ mb: 2 }}>
            <Stack direction="column" spacing={1}>
                <Typography>Cancellation is in progress...</Typography>
                {!forceCancelAvailable && <Typography variant="caption">If the graceful cancellation fails, this run can be force cancelled in {moment(data.forceCancelAvailableAt as moment.MomentInput).fromNow(true)}.</Typography>}
                {forceCancelAvailable && <ForceCancelRunButton fragmentRef={data}/>}
            </Stack>
        </Alert>
    )
}

export default ForceCancelRunAlert
