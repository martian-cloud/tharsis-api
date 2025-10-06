import { Box, CircularProgress } from '@mui/material';
import React, { Suspense } from 'react';

interface Props {
  children: React.ReactNode;
}

function TabContent(props: Props) {
  return (
    <Suspense fallback={<Box
      sx={{
        width: '100%',
        minHeight: 400,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center'
      }}>
      <CircularProgress />
    </Box>}>
      {props.children}
    </Suspense>
  );
}

export default TabContent;
