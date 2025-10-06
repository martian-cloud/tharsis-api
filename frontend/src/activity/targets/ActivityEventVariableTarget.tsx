import { Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { VariableIcon } from '../../common/Icons';
import ActivityEventLink from '../ActivityEventLink';
import ActivityEventListItem from '../ActivityEventListItem';
import { ActivityEventVariableTargetFragment_event$key } from './__generated__/ActivityEventVariableTargetFragment_event.graphql';

const ACTION_TEXT = {
    CREATE: 'created',
    UPDATE: 'updated',
} as any;

interface Props {
    fragmentRef: ActivityEventVariableTargetFragment_event$key
}

function ActivityEventVariableTarget({ fragmentRef }: Props) {
    const data = useFragment<ActivityEventVariableTargetFragment_event$key>(
        graphql`
        fragment ActivityEventVariableTargetFragment_event on ActivityEvent
        {
            action
            namespacePath
            target {
                ...on NamespaceVariable {
                    key
                }
            }
            ...ActivityEventListItemFragment_event
        }
      `, fragmentRef);

    const actionText = ACTION_TEXT[data.action];
    const variable = data.target as any;

    return (
        <ActivityEventListItem
            fragmentRef={data}
            icon={<VariableIcon />}
            primary={<React.Fragment>
                Variable <Typography component="span" sx={{ fontWeight: 500 }}>{variable.key}</Typography> {actionText} in namespace <ActivityEventLink to={`/groups/${data.namespacePath}`}>{data.namespacePath}</ActivityEventLink>
            </React.Fragment>}
        />
    );
}

export default ActivityEventVariableTarget;
