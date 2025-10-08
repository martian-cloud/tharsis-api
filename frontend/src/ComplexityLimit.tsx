import { Box, Typography } from '@mui/material'

function ComplexityLimit() {
    return (
        <Box sx={{ textAlign: 'center' }} padding={4} display="flex" flexDirection="column" justifyContent="center" height="400px">
            <Typography variant="h5">You have made too many requests within a second period.</Typography>
            <Typography variant="h5">Please wait a few moments and then try again.</Typography>
        </Box>
      )
}

export default ComplexityLimit