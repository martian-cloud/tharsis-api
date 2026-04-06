import { Chip } from '@mui/material';
import { useTheme } from '@mui/material/styles';
import React from 'react';
import { Link as RouterLink } from 'react-router-dom';

const STATUS_MAP: Record<string, { label: string }> = {
  applied: { label: 'Complete' },
  apply_queued: { label: 'Apply Queued' },
  applying: { label: 'Applying' },
  canceled: { label: 'Canceled' },
  errored: { label: 'Errored' },
  pending: { label: 'Pending' },
  plan_queued: { label: 'Plan Queued' },
  planned: { label: 'Plan Created' },
  planned_and_finished: { label: 'Complete' },
  planning: { label: 'Planning' },
};

interface Props {
  to: string
  status: string
}

function RunStatusChip(props: Props) {
  const theme = useTheme();
  const entry = STATUS_MAP[props.status];
  const color = entry
    ? theme.palette.runStatus[props.status as keyof typeof theme.palette.runStatus]
    : theme.palette.runStatus.unknown;
  const label = entry?.label ?? 'unknown';

  return (
    <Chip
      to={props.to}
      component={RouterLink}
      clickable
      size="small"
      variant="outlined"
      label={label}
      sx={{ color, borderColor: color, fontWeight: 500 }}
    />
  );
}

export default RunStatusChip;
