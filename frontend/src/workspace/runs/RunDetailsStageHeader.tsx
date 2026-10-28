import { Typography } from '@mui/material';
import Box from '@mui/material/Box';
import ActorDisplay from '../../common/ActorDisplay';
import Timestamp from '../../common/Timestamp';
import { ActorDisplayFragment_actor$key } from '../../common/__generated__/ActorDisplayFragment_actor.graphql';

interface Props {
    stage: 'Plan' | 'Post-Plan' | 'Pre-Plan' | 'Pre-Apply' | 'Post-Apply' | 'Apply';
    triggeredAt?: string | null;
    triggeredByActor?: ActorDisplayFragment_actor$key | null;
}

// RunDetailsStageHeader is the always-visible header row on the plan/apply stage pages,
// naming who triggered the stage and when. Before the stage is triggered (an apply
// awaiting approval) it renders a muted placeholder instead, so the page skeleton
// stays the same across states.
function RunDetailsStageHeader({ stage, triggeredAt, triggeredByActor }: Props) {
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
            {triggeredByActor ? <Typography component="div">
                {stage} triggered{' '}
                {triggeredAt && <Timestamp component="span" timestamp={triggeredAt} />}
                {' '}by{' '}
                <ActorDisplay fragmentRef={triggeredByActor} variant="inline" size={20} />
            </Typography> : <Typography component="div" color="textSecondary">
                {stage} has not been triggered
            </Typography>}
        </Box>
    );
}

export default RunDetailsStageHeader;
