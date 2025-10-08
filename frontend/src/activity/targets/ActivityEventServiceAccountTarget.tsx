import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { ServiceAccountIcon } from '../../common/Icons';
import ActivityEventLink from '../ActivityEventLink';
import ActivityEventListItem from '../ActivityEventListItem';
import { ActivityEventServiceAccountTargetFragment_event$key } from './__generated__/ActivityEventServiceAccountTargetFragment_event.graphql';

const ACTION_TEXT = {
    CREATE: 'created',
    UPDATE: 'updated',
} as any;

interface Props {
    fragmentRef: ActivityEventServiceAccountTargetFragment_event$key
}

function ActivityEventServiceAccountTarget({ fragmentRef }: Props) {
    const data = useFragment<ActivityEventServiceAccountTargetFragment_event$key>(
        graphql`
        fragment ActivityEventServiceAccountTargetFragment_event on ActivityEvent
        {
            action
            namespacePath
            target {
                ...on ServiceAccount {
                    id
                    name
                    description
                    resourcePath
                }
            }
            ...ActivityEventListItemFragment_event
        }
      `, fragmentRef);

    const actionText = ACTION_TEXT[data.action];
    const serviceAccount = data.target as any;

    return (
        <ActivityEventListItem
            fragmentRef={data}
            icon={<ServiceAccountIcon />}
            primary={<React.Fragment>
                Service account <ActivityEventLink to={`/groups/${data.namespacePath}/-/service_accounts/${serviceAccount.id}`}>{serviceAccount.name}</ActivityEventLink> {actionText} in group <ActivityEventLink to={`/groups/${data.namespacePath}`}>{data.namespacePath}</ActivityEventLink>
            </React.Fragment>}
        />
    );
}

export default ActivityEventServiceAccountTarget;
