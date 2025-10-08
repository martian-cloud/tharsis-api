import Typography from '@mui/material/Typography';
import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { VCSProviderIcon } from '../../common/Icons';
import ActivityEventLink from '../ActivityEventLink';
import ActivityEventListItem from '../ActivityEventListItem';
import { ActivityEventVCSProviderTargetFragment_event$key } from './__generated__/ActivityEventVCSProviderTargetFragment_event.graphql';

const ACTION_TEXT = {
    CREATE: 'created',
    UPDATE: 'updated',
} as any;

interface Props {
    fragmentRef: ActivityEventVCSProviderTargetFragment_event$key
}

function ActivityEventVCSProviderTarget({ fragmentRef }: Props) {
    const data = useFragment<ActivityEventVCSProviderTargetFragment_event$key>(
        graphql`
        fragment ActivityEventVCSProviderTargetFragment_event on ActivityEvent
        {
            action
            namespacePath
            target {
                __typename
                ...on VCSProvider {
                    name
                }
            }
            ...ActivityEventListItemFragment_event
        }
      `, fragmentRef);

    const actionText = ACTION_TEXT[data.action];
    const provider = data.target as any;

    return (
        <ActivityEventListItem
            fragmentRef={data}
            icon={<VCSProviderIcon />}
            primary={<React.Fragment>
                VCS Provider <Typography component="span" sx={{ fontWeight: 500 }}>{provider.name}</Typography> {actionText} in group <ActivityEventLink to={`/groups/${data.namespacePath}`}>{data.namespacePath}</ActivityEventLink>
            </React.Fragment>}
        />
    );
}

export default ActivityEventVCSProviderTarget;
