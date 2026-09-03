import graphql from 'babel-plugin-relay/macro';
import humanizeDuration from 'humanize-duration';
import moment from 'moment';
import { useFragment } from 'react-relay/hooks';
import { ResponsiveRow } from '../common/ResponsiveTable';
import Timestamp from '../common/Timestamp';
import Link from '../routes/Link';
import { RunnerJobListItemFragment$key } from './__generated__/RunnerJobListItemFragment.graphql';
import JobStatusChip from '../workspace/runs/JobStatusChip';
import { taskStagePath } from '../workspace/runs/runStageNavigation';

interface Props {
    fragmentRef: RunnerJobListItemFragment$key
}

function RunnerJobListItem({ fragmentRef }: Props) {
    const data = useFragment(graphql`
        fragment RunnerJobListItemFragment on Job {
            id
            status
            type
            # Null for a plan/apply job — those link by type instead, straight to the run's plan or
            # apply page. Set for an OPA job, which has no page of its own: it links to the task
            # stage page that owns its policy check.
            opaData {
                taskStageName
            }
            run {
                id
            }
            timestamps {
                queuedAt
                pendingAt
                runningAt
                finishedAt
            }
            workspace {
                name
                fullPath
            }
            metadata {
                createdAt
                updatedAt
            }
        }
    `, fragmentRef);

    const timestamps = data.timestamps;
    const duration = timestamps?.finishedAt ?
        moment.duration(moment(timestamps.finishedAt as moment.MomentInput).diff(moment(timestamps.runningAt as moment.MomentInput))) : null;

    // A plan/apply job has its own page named by type ("plan"/"apply"). An OPA job does not — it
    // links to the task stage page that owns its policy check instead.
    const runPath = `/groups/${data.workspace.fullPath}/-/runs/${data.run.id}`;
    const taskStageName = data.opaData?.taskStageName;
    const jobLink = taskStageName ? `${runPath}/${taskStagePath(taskStageName)}` : `${runPath}/${data.type}`;

    return (
        <ResponsiveRow cells={[
            { label: 'Status', content: <JobStatusChip status={data.status} to={jobLink} /> },
            { primary: true, content: <Link color="inherit" to={jobLink}>{data.id.substring(0, 8)}...</Link> },
            { label: 'Stage', content: data.type },
            { label: 'Workspace', content: <Link color="inherit" to={`/groups/${data.workspace.fullPath}`}>{data.workspace.name}</Link> },
            { label: 'Duration', content: duration ? humanizeDuration(duration.asMilliseconds()) : '--' },
            { label: 'Created', content: <Timestamp timestamp={data.metadata.createdAt as string} /> },
        ]} />
    );
}

export default RunnerJobListItem;
