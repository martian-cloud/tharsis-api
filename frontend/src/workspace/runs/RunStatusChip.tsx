import { Chip } from '@mui/material';
import blue from '@mui/material/colors/blue';
import green from '@mui/material/colors/green';
import grey from '@mui/material/colors/grey';
import orange from '@mui/material/colors/orange';
import red from '@mui/material/colors/red';
import React from 'react';
import { Link as RouterLink } from 'react-router-dom';

const RUN_STATUS_TYPES = {
  "applied": {
    label: "Complete",
    color: green[500]
  },
  "apply_queued": {
    label: "Apply Queued",
    color: orange[500]
  },
  "applying": {
    label: "Applying",
    color: blue[500]
  },
  "canceled": {
    label: "Canceled",
    color: red[500]
  },
  "errored": {
    label: "Errored",
    color: red[500]
  },
  "pending": {
    label: "Pending",
    color: orange[500]
  },
  "plan_queued": {
    label: "Plan Queued",
    color: orange[500]
  },
  "planned": {
    label: "Plan Created",
    color: green[400]
  },
  "planned_and_finished": {
    label: "Complete",
    color: green[400]
  },
  "planning": {
    label: "Planning",
    color: blue[500]
  }
} as any;

interface Props {
  to: string
  status: string
}

function RunStatusChip(props: Props) {
  const type = RUN_STATUS_TYPES[props.status] ?? { label: 'unknown', color: grey[500] }
  return (
    <Chip
      to={props.to}
      component={RouterLink}
      clickable
      size="small"
      variant="outlined"
      label={type.label}
      sx={{ color: type.color, borderColor: type.color, fontWeight: 500 }}
    />
  );
}

export default RunStatusChip;
