import { Box, FormControlLabel, Switch, Typography } from '@mui/material'

interface Props {
    data: boolean
    onChange: (event: any) => void
}

function PreventDestroyRunSetting(props: Props) {
    const { data, onChange } = props

    return (
        <Box sx={{ mb: 4 }}>
            <Typography variant="subtitle1" gutterBottom>Destroy Infrastructure Protection</Typography>
            <FormControlLabel
                control={<Switch sx={{ m: 2 }}
                    checked={data}
                    color="secondary"
                    onChange={event => onChange(event)}
                />}
                label={data ? "On" : "Off"}
            />
            <Typography variant="subtitle2">When enabled, this will prevent running a destroy plan.</Typography>
        </Box>
    )
}

export default PreventDestroyRunSetting
