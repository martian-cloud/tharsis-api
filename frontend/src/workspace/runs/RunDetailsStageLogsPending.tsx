import { Box, Typography } from '@mui/material';
import Lottie from 'react-lottie-player';
import RocketLottieFileJson from '../../lotties/rocket-in-space-lottie.json';

// RunDetailsStageLogsPending fills the logs tab while the stage's job is pending — a
// runner has claimed it and is preparing to run, so logs will start streaming shortly.
function RunDetailsStageLogsPending() {
    return (
        <Box display="flex" justifyContent="center" sx={{ py: 6 }}>
            <Box display="flex" flexDirection="column" alignItems="center">
                <Lottie
                    renderer={undefined}
                    rendererSettings={undefined}
                    audioFactory={undefined}
                    animationData={RocketLottieFileJson}
                    loop={true}
                    play
                    style={{ width: 250, height: 250 }}
                />
                <Typography variant="h6" align="center">Job is launching and will start shortly</Typography>
            </Box>
        </Box>
    );
}

export default RunDetailsStageLogsPending;
