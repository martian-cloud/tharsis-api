import React from 'react';
import Gravatar from '../common/Gravatar';
import graphql from 'babel-plugin-relay/macro';
import { Box, Stack, TableCell, TableRow } from '@mui/material';
import Timestamp from '../common/Timestamp';
import { useFragment } from 'react-relay/hooks';
import { TeamMemberListItemFragment_member$key } from './__generated__/TeamMemberListItemFragment_member.graphql';

interface Props {
    fragmentRef: TeamMemberListItemFragment_member$key
}

function TeamMemberListItem({ fragmentRef }: Props) {

    const data = useFragment<TeamMemberListItemFragment_member$key>(graphql`
        fragment TeamMemberListItemFragment_member on TeamMember {
            user {
                username
                email
            }
            metadata {
                updatedAt
            }
            isMaintainer
        }
    `, fragmentRef);

    return (
        <TableRow>
            <TableCell sx={{ fontWeight: 'bold' }}>
                <Stack direction="row" alignItems="center" spacing={1}>
                    <React.Fragment>
                        <Gravatar width={24} height={24} sx={{ marginRight: 1 }} email={data.user.email ?? ''} />
                        <Box>{data.user.username}</Box>
                    </React.Fragment>
                </Stack>
            </TableCell>
            <TableCell>
                {data.isMaintainer ? 'Yes' : 'No'}
            </TableCell>
            <TableCell>
                <Timestamp timestamp={data.metadata.updatedAt} />
            </TableCell>
        </TableRow>
    );
}

export default TeamMemberListItem;
