import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { FederatedRegistryIcon } from '../../common/Icons';
import ActivityEventLink from '../ActivityEventLink';
import ActivityEventListItem from '../ActivityEventListItem';
import { ActivityEventFederatedRegistryTargetFragment_event$key } from './__generated__/ActivityEventFederatedRegistryTargetFragment_event.graphql';

const ACTION_TEXT = {
    CREATE: 'created',
    UPDATE: 'updated'
} as any;

interface Props {
    fragmentRef: ActivityEventFederatedRegistryTargetFragment_event$key
}

function ActivityEventFederatedRegistryTarget({ fragmentRef }: Props) {
    const data = useFragment<ActivityEventFederatedRegistryTargetFragment_event$key>(
        graphql`
        fragment ActivityEventFederatedRegistryTargetFragment_event on ActivityEvent
        {
            action
            namespacePath
            target {
                ...on FederatedRegistry {
                    id
                    hostname
                }
            }
            ...ActivityEventListItemFragment_event
        }
      `, fragmentRef);

    const actionText = ACTION_TEXT[data.action];
    const federatedRegistry = data.target as any;

    return (
        <ActivityEventListItem
            fragmentRef={data}
            icon={<FederatedRegistryIcon />}
            primary={<React.Fragment>
                Federated registry <ActivityEventLink to={`/groups/${data.namespacePath}/-/federated_registries/${federatedRegistry.id}`}>{federatedRegistry.hostname}</ActivityEventLink> {actionText} in group <ActivityEventLink to={`/groups/${data.namespacePath}`}>{data.namespacePath}</ActivityEventLink>
            </React.Fragment>}
        />
    );
}

export default ActivityEventFederatedRegistryTarget;
