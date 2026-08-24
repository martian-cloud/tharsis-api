import ApprovalIcon from '@mui/icons-material/AssignmentTurnedInOutlined';
import ChevronRightIcon from '@mui/icons-material/ChevronRight';
import { Badge, Box, ListItemButton, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useLazyLoadQuery } from 'react-relay/hooks';
import { Link as RouterLink } from 'react-router-dom';
import Timestamp from '../common/Timestamp';
import { HomeApprovalsPanelQuery } from './__generated__/HomeApprovalsPanelQuery.graphql';

// The panel shows a total and the newest gate's age, so it takes totalCount from the connection and
// only the single most recent node.
const query = graphql`
    query HomeApprovalsPanelQuery {
        runGatesAwaitingMyDecision(first: 1, sort: CREATED_AT_DESC) {
            totalCount
            edges {
                node {
                    id
                    metadata {
                        createdAt
                    }
                }
            }
        }
    }
`;

function HomeApprovalsPanel() {
    const data = useLazyLoadQuery<HomeApprovalsPanelQuery>(
        query,
        {},
        { fetchPolicy: 'store-and-network' }
    );

    const count = data?.runGatesAwaitingMyDecision?.totalCount ?? 0;
    const latestNode = data?.runGatesAwaitingMyDecision?.edges?.[0]?.node;

    return (
        <ListItemButton component={RouterLink} to="/approval_requests" sx={{ gap: 2 }}>
            <Badge badgeContent={count || null} color="warning">
                <ApprovalIcon color={count > 0 ? 'warning' : undefined} sx={{ width: 28, height: 28 }} />
            </Badge>
            <Box flex={1} minWidth={0}>
                <Typography variant="subtitle1" fontWeight={600}>Awaiting My Approval</Typography>
                <Typography variant="body2" color="textSecondary" noWrap>
                    {latestNode
                        ? <>latest <Timestamp variant="inherit" color="inherit" timestamp={latestNode.metadata.createdAt} /></>
                        : 'No pending approvals'
                    }
                </Typography>
            </Box>
            <ChevronRightIcon sx={{ color: 'text.secondary', flexShrink: 0 }} />
        </ListItemButton>
    );
}

export default HomeApprovalsPanel;
