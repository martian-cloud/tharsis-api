import { Paper, Tooltip, Typography } from '@mui/material';
import Box from '@mui/material/Box';
import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import FailureMessagesList from '../checks/FailureMessagesList';
import CheckStatusChip from '../checks/CheckStatusChip';
import { collectFailureMessages } from '../checks/checks';
import { SummaryRowCol } from './SummaryRow';
import { CheckResultsPanelFragment_checkResult$key } from './__generated__/CheckResultsPanelFragment_checkResult.graphql';

interface Props {
    fragmentRefs: CheckResultsPanelFragment_checkResult$key;
}

function CheckResultsPanel({ fragmentRefs }: Props) {
    const checkResults = useFragment(
        graphql`
        fragment CheckResultsPanelFragment_checkResult on CheckResult @relay(plural: true)
        {
            name
            status
            objects {
                address
                status
                failureMessages
            }
        }
      `, fragmentRefs);

    if (checkResults.length === 0) {
        return null;
    }

    return (
        <Paper variant="outlined" sx={{ marginBottom: 2, pt: 2, px: 2, pb: 0 }}>
            <Typography variant="subtitle2" color="textSecondary" sx={{ mb: 1 }}>Check Results</Typography>
            <Box mx={-2}>
                {checkResults.map((check) => {
                    const failureMessages = collectFailureMessages(check.objects);
                    return (
                        <SummaryRowCol key={check.name} sx={{
                            pl: 2,
                            pr: 2,
                            flexWrap: { xs: 'wrap', md: 'nowrap' },
                            gap: { xs: 0.5, md: 0 },
                        }}>
                            <Box sx={{ width: 75, flexShrink: 0, mr: 1 }}>
                                <CheckStatusChip status={check.status} withBackground />
                            </Box>
                            <Tooltip title={check.name} placement="top">
                                <Typography variant="body2" fontWeight={500} sx={{ fontFamily: 'monospace', whiteSpace: 'nowrap', width: { xs: '100%', md: 250 }, flexShrink: 0, overflow: 'hidden', textOverflow: 'ellipsis' }}>
                                    {check.name}
                                </Typography>
                            </Tooltip>
                            {failureMessages.length > 0 && (
                                <Box sx={{ ml: { xs: 0, md: 1 }, width: { xs: '100%', md: 'auto' } }}>
                                    <FailureMessagesList messages={failureMessages} />
                                </Box>
                            )}
                            <Box component="span" flexGrow={1} />
                        </SummaryRowCol>
                    );
                })}
            </Box>
        </Paper>
    );
}

export default CheckResultsPanel;
