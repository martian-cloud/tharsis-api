import { Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { KeyIcon } from '../../common/Icons';
import ActivityEventLink from '../ActivityEventLink';
import ActivityEventListItem from '../ActivityEventListItem';
import { ActivityEventGPGKeyTargetFragment_event$key } from './__generated__/ActivityEventGPGKeyTargetFragment_event.graphql';

const ACTION_TEXT = {
    CREATE: 'created',
    UPDATE: 'updated',
} as any;

interface Props {
    fragmentRef: ActivityEventGPGKeyTargetFragment_event$key
}

function ActivityEventGPGKeyTarget({ fragmentRef }: Props) {
    const data = useFragment<ActivityEventGPGKeyTargetFragment_event$key>(
        graphql`
        fragment ActivityEventGPGKeyTargetFragment_event on ActivityEvent
        {
            action
            namespacePath
            target {
                ...on GPGKey {
                    id
                    gpgKeyId
                }
            }
            ...ActivityEventListItemFragment_event
        }
      `, fragmentRef);

    const actionText = ACTION_TEXT[data.action];
    const gpgKey = data.target as any;

    return (
        <ActivityEventListItem
            fragmentRef={data}
            icon={<KeyIcon />}
            primary={<React.Fragment>
                GPG key <Typography component="span" sx={{ fontWeight: 500 }}>{gpgKey.gpgKeyId}</Typography> {actionText} in group <ActivityEventLink to={`/groups/${data.namespacePath}`}>{data.namespacePath}</ActivityEventLink>
            </React.Fragment>}
        />
    );
}

export default ActivityEventGPGKeyTarget;
