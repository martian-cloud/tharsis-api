import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { ShieldLockOutline as PolicyIcon } from 'mdi-material-ui';
import ActivityEventLink from '../ActivityEventLink';
import ActivityEventListItem from '../ActivityEventListItem';
import { ActivityEventPolicyTargetFragment_event$key } from './__generated__/ActivityEventPolicyTargetFragment_event.graphql';

const ACTION_TEXT = {
    CREATE: 'created',
    UPDATE: 'updated',
} as any;

interface Props {
    fragmentRef: ActivityEventPolicyTargetFragment_event$key
}

function ActivityEventPolicyTarget({ fragmentRef }: Props) {
    const data = useFragment<ActivityEventPolicyTargetFragment_event$key>(
        graphql`
        fragment ActivityEventPolicyTargetFragment_event on ActivityEvent
        {
            action
            namespacePath
            target {
                ...on Policy {
                    id
                    name
                    groupPath
                }
            }
            ...ActivityEventListItemFragment_event
        }
      `, fragmentRef);

    const actionText = ACTION_TEXT[data.action];
    const policy = data.target as any;

    return (
        <ActivityEventListItem
            fragmentRef={data}
            icon={<PolicyIcon />}
            primary={<React.Fragment>
                Policy <ActivityEventLink to={`/groups/${policy.groupPath}/-/policies/${policy.id}`}>{policy.name}</ActivityEventLink> {actionText} in group <ActivityEventLink to={`/groups/${data.namespacePath}`}>{data.namespacePath}</ActivityEventLink>
            </React.Fragment>}
        />
    );
}

export default ActivityEventPolicyTarget;
