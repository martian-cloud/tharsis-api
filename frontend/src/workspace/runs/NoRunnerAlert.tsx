import { useMemo } from 'react';
import { Alert, AlertTitle, SxProps, Theme } from '@mui/material';
import Link from '../../routes/Link';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { NoRunnerAlertFragment_job$key } from './__generated__/NoRunnerAlertFragment_job.graphql';

interface Props {
    fragmentRef: NoRunnerAlertFragment_job$key;
    sx?: SxProps<Theme>
}

const ALERT_TITLES = {
    NONE: 'No runners are available for this job',
    INACTIVE: 'No runners with an active session are available for this job'
} as const;

const ALERT_MESSAGES = {
    NONE: (settingsPath: string) => (
        <>
            Check <strong>Runner Settings</strong> on the{' '}
            <Link to={settingsPath}>Settings</Link> page to see if this workspace is using tags.
            If the workspace has tags, all of them must be present on the target runner agent
            (note: the runner agent can have additional tags). Also, confirm that the target
            runner agent is enabled.
        </>
    ),
    INACTIVE: (settingsPath: string) => (
        <>
            Check <strong>Runner Agents</strong> to ensure that the target runner agent has an
            active session. Runner agents are targeted by tags so you may also
            need to check <strong>Runner Settings</strong> on the{' '}
            <Link to={settingsPath}>Settings</Link> page as well as the tags for the target
            runner agent.
        </>
    )
} as const;

function NoRunnerAlert ({ fragmentRef, sx }: Props) {
    const data = useFragment<NoRunnerAlertFragment_job$key>(
        graphql`
            fragment NoRunnerAlertFragment_job on Job {
                runnerAvailabilityStatus
                workspace {
                    fullPath
                }
            }
        `,
        fragmentRef
    );

    const workspaceSettingsPath = useMemo(
        () => `/groups/${data.workspace.fullPath}/-/settings`,
        [data.workspace.fullPath]
    );

    const alertMessage = useMemo(() => {
        const messageBuilder = ALERT_MESSAGES[data.runnerAvailabilityStatus as keyof typeof ALERT_MESSAGES];
        return messageBuilder ? messageBuilder(workspaceSettingsPath) : null;
    }, [data.runnerAvailabilityStatus, workspaceSettingsPath]);

    if (!alertMessage) {
        return null;
    }

    return (
        <Alert sx={sx} severity="warning" variant="outlined">
            <AlertTitle>
                {ALERT_TITLES[data.runnerAvailabilityStatus as keyof typeof ALERT_TITLES]}
            </AlertTitle>
            {alertMessage}
        </Alert>
    );
}

export default NoRunnerAlert;
