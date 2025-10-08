import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { RunIcon } from '../../common/Icons';
import ActivityEventLink from '../ActivityEventLink';
import ActivityEventListItem from '../ActivityEventListItem';
import { ActivityEventRunTargetFragment_event$key } from './__generated__/ActivityEventRunTargetFragment_event.graphql';

const ACTION_TEXT = {
    CANCEL: 'canceled',
    CREATE: 'created',
} as any;

interface Props {
    fragmentRef: ActivityEventRunTargetFragment_event$key
}

function ActivityEventRunTarget({ fragmentRef }: Props) {
    const data = useFragment<ActivityEventRunTargetFragment_event$key>(
        graphql`
        fragment ActivityEventRunTargetFragment_event on ActivityEvent
        {
            action
            namespacePath
            target {
                ...on Run {
                    id
                }
            }
            ...ActivityEventListItemFragment_event
        }
      `, fragmentRef);

    const actionText = ACTION_TEXT[data.action];
    const run = data.target as any;

    return (
        <ActivityEventListItem
            fragmentRef={data}
            icon={<RunIcon />}
            primary={<React.Fragment>
                Run <ActivityEventLink
                    to={`/groups/${data.namespacePath}/-/runs/${run.id}`}>{run.id.substring(0, 8)}...
                </ActivityEventLink> {actionText} in <ActivityEventLink to={`/groups/${data.namespacePath}`}>{data.namespacePath}</ActivityEventLink>
            </React.Fragment>}
        />
    );
}

export default ActivityEventRunTarget;
