import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { RunIcon } from '../../common/Icons';
import ActivityEventLink from '../ActivityEventLink';
import ActivityEventListItem from '../ActivityEventListItem';
import { ActivityEventRunTargetFragment_event$key } from './__generated__/ActivityEventRunTargetFragment_event.graphql';

// ACTION_TEXT covers run events keyed directly by action, including the legacy
// CANCEL action retained for historical events. New run updates are recorded as
// UPDATE with an ActivityEventUpdateRunPayload and are rendered from the payload's
// sub-action type (see RUN_UPDATE_TYPE_TEXT).
const ACTION_TEXT = {
    CANCEL: 'canceled',
    CREATE: 'created',
} as any;

// RUN_UPDATE_TYPE_TEXT maps an UPDATE run payload's sub-action type to a verb. These
// keys mirror models.ActivityEventRunUpdateType on the backend.
const RUN_UPDATE_TYPE_TEXT = {
    cancel: 'canceled',
    discard: 'discarded',
    undiscard: 'undiscarded',
    enable_auto_apply: 'auto-apply enabled',
    disable_auto_apply: 'auto-apply disabled',
    retry: 'retried',
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
            payload {
                __typename
                ...on ActivityEventUpdateRunPayload {
                    type
                    nodePath
                }
            }
            ...ActivityEventListItemFragment_event
        }
      `, fragmentRef);

    const payload = data.payload;
    const isRunUpdatePayload = payload?.__typename === 'ActivityEventUpdateRunPayload';
    let actionText = (data.action === 'UPDATE' && isRunUpdatePayload)
        ? (RUN_UPDATE_TYPE_TEXT[payload.type] ?? 'updated')
        : (ACTION_TEXT[data.action] ?? 'updated');
    // Node-scoped updates (e.g. retry) carry the node path so we can name the node acted on.
    if (isRunUpdatePayload && payload.nodePath) {
        actionText = `${actionText} (${payload.nodePath})`;
    }
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
