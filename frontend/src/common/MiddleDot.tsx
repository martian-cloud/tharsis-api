import { alpha, useTheme } from '@mui/material';
import Typography from '@mui/material/Typography';
import React from 'react';

function MiddleDot() {
  const theme = useTheme();
  return (
    <Typography sx={{ margin: '0 8px', color: alpha(theme.palette.text.primary, 0.5) }} display="inline" >
      &bull;
    </Typography>
  );
}

export default MiddleDot;
