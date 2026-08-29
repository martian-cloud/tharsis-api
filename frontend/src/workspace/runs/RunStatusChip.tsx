import WarningIcon from '@mui/icons-material/Error';
import { Chip } from '@mui/material';
import Tooltip from '@mui/material/Tooltip';
import { useTheme } from '@mui/material/styles';
import React from 'react';
import { Link as RouterLink } from 'react-router-dom';

export const STATUS_MAP: Record<string, { label: string }> = {
  applied: { label: 'Complete' },
  apply_queued: { label: 'Apply Queued' },
  apply_queuing: { label: 'Waiting to be queued' },
  applying: { label: 'Applying' },
  canceled: { label: 'Canceled' },
  discarded: { label: 'Discarded' },
  errored: { label: 'Errored' },
  pending: { label: 'Waiting' },
  plan_queued: { label: 'Plan Queued' },
  plan_queuing: { label: 'Waiting to be queued' },
  planned: { label: 'Plan Created' },
  planned_and_finished: { label: 'Complete' },
  planning: { label: 'Planning' },
  pre_plan_queuing: { label: 'Waiting to be queued' },
  pre_plan_running: { label: 'Pre Planning' },
  pre_plan_awaiting_decision: { label: 'Awaiting Approval' },
  pre_plan_completed: { label: 'Pre-Plan Complete' },
  post_plan_running: { label: 'Post Planning' },
  post_plan_awaiting_decision: { label: 'Awaiting Approval' },
  pre_apply_queuing: { label: 'Waiting to be queued' },
  pre_apply_running: { label: 'Pre Applying' },
  pre_apply_awaiting_decision: { label: 'Awaiting Approval' },
  pre_apply_completed: { label: 'Pre-Apply Complete' },
  post_apply_running: { label: 'Post Applying' },
};

const ADVISORY_FAILURES_TOOLTIP = "One or more advisory policies failed.";

interface Props {
  // When set, the chip renders as a link to this path; otherwise it is a plain
  // (non-clickable) status indicator.
  to?: string
  status: string
  // Adds a warning glyph inside the chip, left of the label. An advisory policy failure never blocks a
  // run, so the status label alone cannot express one — the glyph qualifies the status rather than
  // standing beside it as a second chip competing for the eye.
  hasAdvisoryFailures: boolean
}

function RunStatusChip(props: Props) {
  const theme = useTheme();
  const entry = STATUS_MAP[props.status];
  const color = entry
    ? theme.palette.runStatus[props.status.toLowerCase() as keyof typeof theme.palette.runStatus]
    : theme.palette.runStatus.unknown;
  const label = entry?.label ?? 'unknown';
  const sx = {
    color,
    borderColor: color,
    fontWeight: 500,
    // Amber regardless of the status color: the glyph is the advisory signal, the chip is still the status.
    ...(props.hasAdvisoryFailures && {
      '& .MuiChip-icon': { color: 'warning.main', width: 14, height: 14, marginLeft: .5 }
    }),
  };
  const icon = props.hasAdvisoryFailures ? <WarningIcon /> : undefined;

  const chip = !props.to ? (
    <Chip icon={icon} size="small" variant="outlined" label={label} sx={sx} />
  ) : (
    <Chip
      to={props.to}
      component={RouterLink}
      clickable
      icon={icon}
      size="small"
      variant="outlined"
      label={label}
      sx={sx}
    />
  );

  if (!props.hasAdvisoryFailures) {
    return chip;
  }

  // The whole chip is the hover target rather than the glyph alone, so the qualifier and what it qualifies
  // explain themselves together.
  return <Tooltip title={ADVISORY_FAILURES_TOOLTIP}>{chip}</Tooltip>;
}

export default RunStatusChip;
