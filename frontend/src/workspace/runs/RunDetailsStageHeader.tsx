import { Typography } from '@mui/material';
import Box from '@mui/material/Box';
import Gravatar from '../../common/Gravatar';
import Timestamp from '../../common/Timestamp';

interface Props {
    stage: 'Plan' | 'Apply';
    triggeredAt?: string | null;
    triggeredBy?: string | null;
}

// RunDetailsStageHeader is the always-visible header row on the plan/apply stage pages,
// naming who triggered the stage and when. Before the stage is triggered (an apply
// awaiting approval) it renders a muted placeholder instead, so the page skeleton
// stays the same across states.
function RunDetailsStageHeader({ stage, triggeredAt, triggeredBy }: Props) {
    return (
        <Box
            sx={{
                paddingTop: 1,
                marginBottom: 2,
                display: 'flex',
                flexDirection: { xs: 'column', md: 'row' },
                alignItems: { xs: 'flex-start', md: 'center' },
                justifyContent: { xs: 'flex-start', md: 'space-between' },
                gap: { xs: 1 },
            }}>
            {triggeredBy ? <Typography component="div">
                {stage} triggered{' '}
                {triggeredAt && <Timestamp component="span" timestamp={triggeredAt} />}
                {' '}by{' '}
                <Gravatar sx={{ display: 'inline-block', verticalAlign: 'middle' }} width={20} height={20} email={triggeredBy} />
                {' '}{triggeredBy}
            </Typography> : <Typography component="div" color="textSecondary">
                {stage} has not been triggered
            </Typography>}
        </Box>
    );
}

export default RunDetailsStageHeader;
