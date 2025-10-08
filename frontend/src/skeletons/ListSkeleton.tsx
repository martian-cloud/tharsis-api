import { Box, Skeleton } from '@mui/material';
import React, { useEffect, useState } from 'react';

interface Props {
  delay?: number
  rowCount?: number
  size?: 'small' | 'medium' | 'large'
}

const DEFAULT_DELAY = 300;

const HEIGHT_MAP = {
  'small': 32,
  'medium': 56,
  'large': 72
}

function ListSkeleton(props: Props) {
  const delay = props.delay ?? DEFAULT_DELAY;
  const [ready, setReady] = useState(delay === 0);

  const height = HEIGHT_MAP[props.size ?? 'medium']

  useEffect(() => {
    let timeout: NodeJS.Timeout | null;
    if (!ready) {
      timeout = setTimeout(() => setReady(true), delay);
    }
    return () => {
      if (timeout) clearTimeout(timeout);
    };
  }, [delay, ready]);

  return ready ? (
    <Box>
      {Array.from(Array(props.rowCount ?? 1)).map((item: any, index: number) => <Skeleton key={index} height={height} />)}
    </Box>
  ) : null;
}

export default ListSkeleton;
