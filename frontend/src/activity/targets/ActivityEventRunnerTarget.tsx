import { Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { RunnerIcon } from '../../common/Icons';
import ActivityEventLink from '../ActivityEventLink';
import ActivityEventListItem from '../ActivityEventListItem';
import { ActivityEventRunnerTargetFragment_event$key } from './__generated__/ActivityEventRunnerTargetFragment_event.graphql';

const ACTION_TEXT = {
    CREATE: 'created',
    UPDATE: 'updated',
} as any;

interface Props {
    fragmentRef: ActivityEventRunnerTargetFragment_event$key
}

function ActivityEventRunnerTarget({ fragmentRef }: Props) {
    const data = useFragment<ActivityEventRunnerTargetFragment_event$key>(
        graphql`
        fragment ActivityEventRunnerTargetFragment_event on ActivityEvent
        {
            action
            namespacePath
            target {
                ...on Runner {
                    name
                }
            }
            ...ActivityEventListItemFragment_event
        }
      `, fragmentRef);

    const actionText = ACTION_TEXT[data.action];
    const runner = data.target as any;

    return (
        <ActivityEventListItem
            fragmentRef={data}
            icon={<RunnerIcon />}
            primary={<React.Fragment>
                Runner agent <Typography component="span" sx={{ fontWeight: 500 }}>{runner.name}</Typography> {actionText} in group <ActivityEventLink to={`/groups/${data.namespacePath}`}>{data.namespacePath}</ActivityEventLink>
                </React.Fragment>}
        />
    );
}

export default ActivityEventRunnerTarget;
