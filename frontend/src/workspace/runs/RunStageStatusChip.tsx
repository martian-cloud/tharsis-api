import { Chip } from '@mui/material';
import grey from '@mui/material/colors/grey';
import React from 'react';
import RunStageStatusTypes from './RunStageStatusTypes';

interface Props {
  status: string
}

function RunStageStatusChip(props: Props) {
  const type = RunStageStatusTypes[props.status] ?? { label: 'unknown', color: grey[500] }
  const StatusIcon = type.icon;
  return (
    <Chip
      icon={<StatusIcon />}
      size="small"
      variant="outlined"
      label={type.label}
      sx={{
        color: type.color,
        borderColor: type.color,
        fontWeight: 500,
        '& .MuiChip-icon': {
          color: type.color,
          marginLeft: .5
        }
      }}
    />
  );
}

export default RunStageStatusChip;
