import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { Chip, Collapse } from '@mui/material';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import React, { useState } from 'react';

const MAX_VISIBLE = 3;

interface Props {
    messages: readonly string[];
}

function MessageRow({ message }: { message: string }) {
    return (
        <Box sx={{ display: 'flex', alignItems: 'flex-start', py: 0.5 }}>
            <ErrorOutlineIcon sx={{ width: 14, height: 14, color: 'error.main', mr: 0.5, mt: '2px', flexShrink: 0 }} />
            <Typography variant="body2" color="error">
                {message}
            </Typography>
        </Box>
    );
}

function FailureMessagesList({ messages }: Props) {
    const [expanded, setExpanded] = useState(false);

    if (messages.length === 0) {
        return null;
    }

    // Show up to MAX_VISIBLE messages directly.
    if (messages.length <= MAX_VISIBLE) {
        return (
            <Box>
                {messages.map((msg, index) => (
                    <MessageRow key={index} message={msg} />
                ))}
            </Box>
        );
    }

    // More than MAX_VISIBLE: show only the chip, expand to reveal all messages.
    return (
        <Box>
            <Chip
                label={expanded ? 'Show less' : `${messages.length} failures`}
                size="small"
                color="error"
                variant="outlined"
                onClick={() => setExpanded(!expanded)}
                icon={<ExpandMoreIcon sx={{ transform: expanded ? 'rotate(180deg)' : 'none', transition: '0.2s' }} />}
                sx={{ cursor: 'pointer' }}
            />
            <Collapse in={expanded}>
                <Box sx={{ mt: 0.5 }}>
                    {messages.map((msg, index) => (
                        <MessageRow key={index} message={msg} />
                    ))}
                </Box>
            </Collapse>
        </Box>
    );
}

export default FailureMessagesList;
