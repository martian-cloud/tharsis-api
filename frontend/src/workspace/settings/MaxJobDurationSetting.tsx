import React from 'react'
import { Box, Typography, TextField } from '@mui/material'
import { useFragment } from 'react-relay/hooks'
import graphql from 'babel-plugin-relay/macro'
import { MaxJobDurationSettingFragment_workspace$key } from './__generated__/MaxJobDurationSettingFragment_workspace.graphql'

interface Props {
    fragmentRef: MaxJobDurationSettingFragment_workspace$key
    data: any
    onChange: (event: any) => void
}

function MaxJobDurationSetting(props: Props) {
    const { data, onChange } = props

    const maxJobData = useFragment(
        graphql`
        fragment MaxJobDurationSettingFragment_workspace on Workspace
        {
            maxJobDuration
        }
    `, props.fragmentRef
    )

    const convertJobTime = (num: number) => {
        const hours = Math.floor(num / 60)
        const min = num % 60
        return `${hours >= 1 ? `${hours} hour${hours === 1 ? '' : 's'} and ` : ''} ${min} minute${min === 1 ? '' : 's'}`
    }

    return (
        <Box sx={{ mb: 4 }}>
            <Typography variant="subtitle1" gutterBottom>Maximum Job Duration</Typography>
            <Typography marginBottom={2} variant="subtitle2">Current Maximum Job Duration: {convertJobTime(maxJobData.maxJobDuration)}</Typography>
            <Box>
                <TextField
                    sx={{ minWidth: 250, mb: 1 }}
                    size="small"
                    type="number"
                    value={data}
                    onChange={event => onChange(event)}
                />
                <Typography sx={{ mb: 1 }} variant="subtitle2">Jobs will timeout if they run longer than the maximum job duration. Input value is in minutes by default.</Typography>
            </Box>
        </Box>
    )
}

export default MaxJobDurationSetting
