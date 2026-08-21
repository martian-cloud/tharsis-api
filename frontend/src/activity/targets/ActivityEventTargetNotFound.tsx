import React from 'react';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { UnavailableResourceIcon } from '../../common/Icons';
import ActivityEventListItem from '../ActivityEventListItem';
import { ActivityEventTargetNotFoundFragment_event$key } from './__generated__/ActivityEventTargetNotFoundFragment_event.graphql';

interface Props {
    fragmentRef: ActivityEventTargetNotFoundFragment_event$key
}

function ActivityEventTargetNotFound({ fragmentRef }: Props) {
    const data = useFragment<ActivityEventTargetNotFoundFragment_event$key>(graphql`
        fragment ActivityEventTargetNotFoundFragment_event on ActivityEvent {
            targetType
            ...ActivityEventListItemFragment_event
        }
      `, fragmentRef);

    // Deliberately not phrased as a deletion. The target resolves to null whenever the API returns
    // ENotFound for it, and a user with no viewer access to the owning namespace gets exactly that
    // code rather than EForbidden (see UserCaller.UnauthorizedError), so the far more common cause is
    // access to a live resource having been revoked -- the resource is still there for everyone else.
    // The two are indistinguishable from here, which is why this says nothing about which happened.
    return (
        <ActivityEventListItem
            fragmentRef={data}
            icon={<UnavailableResourceIcon />}
            primary={<React.Fragment>This event can't be displayed because the resource is no longer accessible</React.Fragment>}
        />
    );
}

export default ActivityEventTargetNotFound;
