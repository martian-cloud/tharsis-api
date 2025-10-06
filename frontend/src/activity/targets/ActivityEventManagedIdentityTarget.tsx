import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { ManagedIdentityIcon } from '../../common/Icons';
import ActivityEventLink from '../ActivityEventLink';
import ActivityEventListItem from '../ActivityEventListItem';
import { ActivityEventManagedIdentityTargetFragment_event$key } from './__generated__/ActivityEventManagedIdentityTargetFragment_event.graphql';

const ACTION_TEXT = {
    ADD: 'assigned to',
    CREATE: 'created',
    REMOVE: 'unassigned from',
    UPDATE: 'updated',
    MIGRATE: 'moved from',
} as any;

interface Props {
    fragmentRef: ActivityEventManagedIdentityTargetFragment_event$key
}

function ActivityEventManagedIdentityTarget({ fragmentRef }: Props) {
    const data = useFragment<ActivityEventManagedIdentityTargetFragment_event$key>(
        graphql`
        fragment ActivityEventManagedIdentityTargetFragment_event on ActivityEvent
        {
            action
            namespacePath
            target {
                ...on ManagedIdentity {
                    id
                    name
                    description
                    resourcePath
                }
            }
            payload {
                __typename
                ...on ActivityEventMoveManagedIdentityPayload {
                    previousGroupPath
                }
            }
            ...ActivityEventListItemFragment_event
        }
      `, fragmentRef);

    const actionText = ACTION_TEXT[data.action];
    const managedIdentity = data.target as any;
    const payload = data.payload as any;

    const identityLink = <ActivityEventLink to={`/groups/${data.namespacePath}/-/managed_identities/${managedIdentity.id}`}>{managedIdentity.name}</ActivityEventLink>;
    const namespaceLink = <ActivityEventLink to={`/groups/${data.namespacePath}`}>{data.namespacePath}</ActivityEventLink>;

    let primary;
    switch (data.action) {
        case 'CREATE':
        case 'UPDATE':
            primary = <React.Fragment>Managed identity {identityLink} {actionText} in group {namespaceLink}</React.Fragment>;
            break;
        case 'ADD':
        case 'REMOVE':
            primary = <React.Fragment> Managed identity {identityLink} {actionText} workspace {namespaceLink}</React.Fragment>
            break;
        case 'DELETE_CHILD_RESOURCE':
            primary = <React.Fragment>Managed identity access rule removed from managed identity {identityLink}</React.Fragment>
            break;
        case 'MIGRATE': {
            primary = <React.Fragment>Managed identity {identityLink} {actionText} group <ActivityEventLink to={`/groups/${payload?.previousGroupPath}`}>{payload?.previousGroupPath}</ActivityEventLink></React.Fragment>;
            break;
        }
    }

    return (
        <ActivityEventListItem
            fragmentRef={data}
            icon={<ManagedIdentityIcon />}
            primary={primary}
        />
    );
}

export default ActivityEventManagedIdentityTarget;
