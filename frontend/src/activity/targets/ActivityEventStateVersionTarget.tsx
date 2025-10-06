import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { StateVersionIcon } from '../../common/Icons';
import ActivityEventLink from '../ActivityEventLink';
import ActivityEventListItem from '../ActivityEventListItem';
import { ActivityEventStateVersionTargetFragment_event$key } from './__generated__/ActivityEventStateVersionTargetFragment_event.graphql';

const ACTION_TEXT = {
    CREATE: 'created',
} as any;

interface Props {
    fragmentRef: ActivityEventStateVersionTargetFragment_event$key
}

function ActivityEventStateVersionTarget({ fragmentRef }: Props) {
    const data = useFragment<ActivityEventStateVersionTargetFragment_event$key>(
        graphql`
        fragment ActivityEventStateVersionTargetFragment_event on ActivityEvent
        {
            action
            namespacePath
            target {
                __typename
                ...on StateVersion {
                    id
                }
            }
            ...ActivityEventListItemFragment_event
        }
      `, fragmentRef);

    const actionText = ACTION_TEXT[data.action];
    const stateVersion = data.target as any;

    return (
        <ActivityEventListItem
            fragmentRef={data}
            icon={<StateVersionIcon />}
            primary={<React.Fragment>
                State version <ActivityEventLink to={`/groups/${data.namespacePath}/-/state_versions/${stateVersion.id}`}>{stateVersion.id.substring(0, 8)}...</ActivityEventLink> {actionText} in workspace <ActivityEventLink to={`/groups/${data.namespacePath}`}>{data.namespacePath}</ActivityEventLink>
            </React.Fragment>}
        />
    );
}

export default ActivityEventStateVersionTarget;
