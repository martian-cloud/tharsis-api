import { Chip, Tooltip } from '@mui/material';
import ListItem from '@mui/material/ListItem';
import ListItemText from '@mui/material/ListItemText';
import graphql from 'babel-plugin-relay/macro';
import moment from 'moment';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { Link as LinkRouter } from 'react-router-dom';
import { StateVersionDependencyListItemFragment_dependency$key } from './__generated__/StateVersionDependencyListItemFragment_dependency.graphql';

const statusMap = {
    deleted: {
        color: 'error',
        tooltip: 'This dependency is referencing a workspace that has been deleted'
    },
    stale: {
        color: 'warning',
        tooltip: 'This dependency is referencing an older version of the workspace state'
    }
} as any;

interface Props {
    fragmentRef: StateVersionDependencyListItemFragment_dependency$key
}

function StateVersionDependencyListItem(props: Props) {
    const data = useFragment<StateVersionDependencyListItemFragment_dependency$key>(
        graphql`
        fragment StateVersionDependencyListItemFragment_dependency on StateVersionDependency
        {
            workspacePath
            stateVersion {
                id
                metadata {
                    updatedAt
                }
            }
            workspace {
                id
                currentStateVersion {
                    id
                }
            }
        }
      `, props.fragmentRef);

    const status = !data.workspace ? 'deleted' : data.stateVersion?.id === data.workspace.currentStateVersion?.id ? 'latest' : 'stale';

    return (
        <ListItem button component={LinkRouter} to={`/groups/${data.workspacePath}`} divider>
            <ListItemText
                primary={data.workspacePath}
                secondary={data.stateVersion ? `updated ${moment(data.stateVersion.metadata.updatedAt as moment.MomentInput).fromNow()}` : ''}
            />
            {status !== 'latest' && <Tooltip title={statusMap[status].tooltip}>
                <Chip label={status} color={statusMap[status].color} />
            </Tooltip>}
        </ListItem>
    );
}

export default StateVersionDependencyListItem;