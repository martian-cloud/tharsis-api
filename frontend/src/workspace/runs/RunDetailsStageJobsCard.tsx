import { Link as MuiLink, Paper, Typography } from '@mui/material';

interface Props {
    stage: 'plan' | 'apply';
    totalCount: number;
    onOpenJobs: () => void;
}

// RunDetailsStageJobsCard is the always-visible jobs summary card on the plan/apply
// stage pages. The count opens the jobs dialog; with no jobs yet there is nothing to
// show, so the count renders as plain text instead of a link.
function RunDetailsStageJobsCard({ stage, totalCount, onOpenJobs }: Props) {
    const countText = `${totalCount} job${totalCount === 1 ? '' : 's'}`;

    return (
        <Paper variant="outlined" sx={{ padding: 2, marginBottom: 2 }}>
            <Typography variant="body2" component="div">This {stage} has
                {totalCount > 0 ? <MuiLink
                    component="button"
                    color="secondary"
                    underline="hover"
                    onClick={onOpenJobs}
                    variant="body2"
                    sx={{ marginLeft: '4px', fontWeight: 600 }}
                >{countText}
                </MuiLink> : <Typography component="span" variant="body2" sx={{ marginLeft: '4px', fontWeight: 600 }}>
                    {countText}
                </Typography>}
            </Typography>
        </Paper>
    );
}

export default RunDetailsStageJobsCard;
