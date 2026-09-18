import { Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { ShieldAccountOutline as RoleBindingIcon } from 'mdi-material-ui';
import ActivityEventLink from '../ActivityEventLink';
import ActivityEventListItem from '../ActivityEventListItem';
import { ActivityEventWorkspaceRoleBindingTargetFragment_event$key } from './__generated__/ActivityEventWorkspaceRoleBindingTargetFragment_event.graphql';

const ACTION_TEXT = {
    CREATE: 'bound',
    UPDATE: 'changed',
} as any;

interface Props {
    fragmentRef: ActivityEventWorkspaceRoleBindingTargetFragment_event$key
}

function ActivityEventWorkspaceRoleBindingTarget({ fragmentRef }: Props) {
    const data = useFragment<ActivityEventWorkspaceRoleBindingTargetFragment_event$key>(
        graphql`
        fragment ActivityEventWorkspaceRoleBindingTargetFragment_event on ActivityEvent
        {
            action
            target {
                ...on WorkspaceRoleBinding {
                    workspace {
                        name
                        fullPath
                    }
                    role {
                        name
                    }
                }
            }
            ...ActivityEventListItemFragment_event
        }
      `, fragmentRef);

    const actionText = ACTION_TEXT[data.action] ?? 'updated';
    const target = data.target as any;

    const workspaceLink = <ActivityEventLink to={`/groups/${target.workspace.fullPath}`}>{target.workspace.name}</ActivityEventLink>;
    const roleName = target?.role?.name;

    const primary = roleName
        ? <React.Fragment>Workspace {workspaceLink} role binding {actionText} to <Typography component="span" sx={{ fontWeight: 500 }}>{roleName}</Typography></React.Fragment>
        : <React.Fragment>Workspace {workspaceLink} role binding {actionText}</React.Fragment>;

    return (
        <ActivityEventListItem
            fragmentRef={data}
            icon={<RoleBindingIcon />}
            primary={primary}
        />
    );
}

export default ActivityEventWorkspaceRoleBindingTarget;
