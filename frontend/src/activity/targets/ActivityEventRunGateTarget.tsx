import { alpha, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { RunIcon } from '../../common/Icons';
import ActivityEventLink from '../ActivityEventLink';
import ActivityEventListItem from '../ActivityEventListItem';
import { ActivityEventRunGateTargetFragment_event$key } from './__generated__/ActivityEventRunGateTargetFragment_event.graphql';

// Keyed by the ActivityEventUpdateRunGatePayloadType enum values, which are upper-case on the wire --
// unlike the run payload's type, which is a plain String carrying the model's lower-case constant.
const RUN_GATE_UPDATE_TYPE_TEXT: Record<string, string> = {
    APPROVE: 'approved',
    REJECT: 'rejected',
    OVERRIDE: 'overridden',
};

interface Props {
    fragmentRef: ActivityEventRunGateTargetFragment_event$key
}

function ActivityEventRunGateTarget({ fragmentRef }: Props) {
    const theme = useTheme();
    const data = useFragment<ActivityEventRunGateTargetFragment_event$key>(
        graphql`
        fragment ActivityEventRunGateTargetFragment_event on ActivityEvent
        {
            namespacePath
            target {
                ...on RunGate {
                    id
                    run {
                        id
                    }
                }
            }
            payload {
                __typename
                ...on ActivityEventUpdateRunGatePayload {
                    # Aliased because ActivityEventUpdateRunPayload also has a "type", of a different
                    # GraphQL type, and both payload fragments are spread into the same
                    # ActivityEvent.payload selection in ActivityEventList.
                    gateUpdateType: type
                    comment
                }
            }
            ...ActivityEventListItemFragment_event
        }
      `, fragmentRef);

    const payload = data.payload;
    const isRunGatePayload = payload?.__typename === 'ActivityEventUpdateRunGatePayload';
    const actionText = isRunGatePayload
        ? (RUN_GATE_UPDATE_TYPE_TEXT[payload.gateUpdateType] ?? 'updated')
        : 'updated';
    const gate = data.target as any;
    const comment = isRunGatePayload ? payload.comment : null;

    // The comment is the decider's own words, so it is quoted on its own line under the sentence rather
    // than run into it, where a long one pushed the namespace link off the end of the row. Styled to
    // match how the same comment reads on the run's policy panel, so a decision looks the same in both
    // places it is shown.
    const commentBlock = comment ? (
        <Typography
            variant="body2"
            sx={{
                minWidth: 0,
                color: theme.palette.text.secondary,
                borderLeft: `2px solid ${alpha(theme.palette.common.white, 0.23)}`,
                paddingLeft: '10px',
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-word',
            }}
        >
            {comment}
        </Typography>
    ) : undefined;

    return (
        <ActivityEventListItem
            fragmentRef={data}
            icon={<RunIcon />}
            primary={<React.Fragment>
                Run <ActivityEventLink
                    to={`/groups/${data.namespacePath}/-/runs/${gate.run?.id}`}>{gate.run?.id?.substring(0, 8)}
                </ActivityEventLink> gate <Typography component="span" sx={{ fontWeight: 500 }}>{actionText}</Typography> in <ActivityEventLink to={`/groups/${data.namespacePath}`}>{data.namespacePath}</ActivityEventLink>
            </React.Fragment>}
            secondary={commentBlock}
        />
    );
}

export default ActivityEventRunGateTarget;
