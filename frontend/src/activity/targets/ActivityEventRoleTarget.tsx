import { Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { RoleIcon } from '../../common/Icons';
import ActivityEventListItem from '../ActivityEventListItem';
import { ActivityEventRoleTargetFragment_event$key } from './__generated__/ActivityEventRoleTargetFragment_event.graphql';

const ACTION_TEXT = {
    CREATE: 'created',
    UPDATE: 'updated',
} as any;

interface Props {
    fragmentRef: ActivityEventRoleTargetFragment_event$key
}

function ActivityEventRoleTarget({ fragmentRef }: Props) {
    const data = useFragment<ActivityEventRoleTargetFragment_event$key>(
        graphql`
        fragment ActivityEventRoleTargetFragment_event on ActivityEvent
        {
            action
            target {
                ...on Role {
                    name
                }
            }
            ...ActivityEventListItemFragment_event
        }
      `, fragmentRef);

    const actionText = ACTION_TEXT[data.action];
    const role = data.target as any;

    return (
        <ActivityEventListItem
            fragmentRef={data}
            icon={<RoleIcon />}
            primary={<React.Fragment>Role <Typography component="span" sx={{ fontWeight: 500 }}>{role.name}</Typography> {actionText}</React.Fragment>}
        />
    );
}

export default ActivityEventRoleTarget;
