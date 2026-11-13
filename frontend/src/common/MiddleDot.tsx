import { alpha, useTheme } from '@mui/material';
import Typography from '@mui/material/Typography';
import React from 'react';

function MiddleDot() {
  const theme = useTheme();
  return (
    <Typography
      sx={{
        display: "inline",
        margin: '0 8px',
        color: alpha(theme.palette.text.primary, 0.5)
      }}>•
          </Typography>
  );
}

export default MiddleDot;
