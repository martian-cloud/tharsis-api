import { Chip } from '@mui/material';
import { useTheme } from '@mui/material/styles';
import React from 'react';
import { Link as RouterLink } from 'react-router-dom';

const STATUS_MAP: Record<string, { label: string }> = {
  applied: { label: 'Complete' },
  apply_queued: { label: 'Apply Queued' },
  applying: { label: 'Applying' },
  canceled: { label: 'Canceled' },
  discarded: { label: 'Discarded' },
  errored: { label: 'Errored' },
  pending: { label: 'Waiting' },
  plan_queued: { label: 'Plan Queued' },
  planned: { label: 'Plan Created' },
  planned_and_finished: { label: 'Complete' },
  planning: { label: 'Planning' },
  queuing: { label: 'Queuing' },
  queuing_apply: { label: 'Apply Queuing' },
};

interface Props {
  // When set, the chip renders as a link to this path; otherwise it is a plain
  // (non-clickable) status indicator.
  to?: string
  status: string
}

function RunStatusChip(props: Props) {
  const theme = useTheme();
  const entry = STATUS_MAP[props.status];
  const color = entry
    ? theme.palette.runStatus[props.status as keyof typeof theme.palette.runStatus]
    : theme.palette.runStatus.unknown;
  const label = entry?.label ?? 'unknown';
  const sx = { color, borderColor: color, fontWeight: 500 };

  if (!props.to) {
    return <Chip size="small" variant="outlined" label={label} sx={sx} />;
  }

  return (
    <Chip
      to={props.to}
      component={RouterLink}
      clickable
      size="small"
      variant="outlined"
      label={label}
      sx={sx}
    />
  );
}

export default RunStatusChip;
