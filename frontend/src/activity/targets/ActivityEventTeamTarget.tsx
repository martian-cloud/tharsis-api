import { Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { TeamIcon } from '../../common/Icons';
import ActivityEventListItem from '../ActivityEventListItem';
import { ActivityEventTeamTargetFragment_event$key } from './__generated__/ActivityEventTeamTargetFragment_event.graphql';

interface Props {
    fragmentRef: ActivityEventTeamTargetFragment_event$key
}

function ActivityEventTeamTarget({ fragmentRef }: Props) {
    const data = useFragment<ActivityEventTeamTargetFragment_event$key>(
        graphql`
        fragment ActivityEventTeamTargetFragment_event on ActivityEvent
        {
            action
            target {
                ...on Team {
                    name
                }
            }
            payload {
                __typename
                ... on ActivityEventUpdateTeamMemberPayload {
                    user {
                        username
                    }
                    maintainer
                 }
                 ... on ActivityEventRemoveTeamMemberPayload {
                    user {
                        username
                    }
                }
                ... on ActivityEventAddTeamMemberPayload {
                    user {
                        username
                    }
                    maintainer
                }
            }
            ...ActivityEventListItemFragment_event
        }
      `, fragmentRef);

    const team = data.target as any;
    const payload = data.payload as any;

    let primary;
    switch (data.action) {
        case 'CREATE':
            primary = (
                <React.Fragment>
                    Team <Typography component="span" sx={{ fontWeight: 500 }}>{team.name}</Typography> created
                </React.Fragment>
            );
            break;
        case 'ADD_MEMBER':
            primary = (
                <React.Fragment>
                    User <Typography component="span" sx={{ fontWeight: 500 }}>{payload.user?.username || 'unknown'}</Typography> added to team <Typography component="span" sx={{ fontWeight: 500 }}>{team.name}</Typography>
                </React.Fragment>
            );
            break;
        case 'REMOVE_MEMBER':
            primary = (
                <React.Fragment>
                    User <Typography component="span" sx={{ fontWeight: 500 }}>{payload.user?.username || 'unknown'}</Typography> removed from team <Typography component="span" sx={{ fontWeight: 500 }}>{team.name}</Typography> 
                </React.Fragment>
            );
            break;
        case 'UPDATE_MEMBER':
            primary = (
                <React.Fragment>
                    Team member <Typography component="span" sx={{ fontWeight: 500 }}>{payload.user?.username || 'unknown'}</Typography> maintainer status changed to {payload.maintainer ? 'true' : 'false'} for team <Typography component="span" sx={{ fontWeight: 500 }}>{team.name}</Typography>
                </React.Fragment>
            );
            break;
    }

    return (
        <ActivityEventListItem
            fragmentRef={data}
            icon={<TeamIcon />}
            primary={primary}
        />
    );
}

export default ActivityEventTeamTarget;
