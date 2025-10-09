import { Box, styled } from '@mui/material';
import React from 'react';

interface StarProps {
  size: number;
  x: number;
  y: number;
  animationDuration: number;
}

const SpaceContainer = styled(Box)({
  position: 'fixed',
  width: '100vw',
  height: '100vh',
  overflow: 'hidden',
  zIndex: -1,
});

const Star = styled('div')<StarProps>(({ size, x, y, animationDuration }) => ({
  position: 'absolute',
  backgroundColor: '#ffffff',
  width: `${size}px`,
  height: `${size}px`,
  borderRadius: '50%',
  left: `${x}%`,
  top: `${y}%`,
  animation: `twinkle ${animationDuration}s infinite alternate`,
  '@keyframes twinkle': {
    '0%': { opacity: 0.2 },
    '100%': { opacity: 1 },
  },
}));

// Generate random stars
const generateStars = (count: number): JSX.Element[] => {
  const stars: JSX.Element[] = [];
  for (let i = 0; i < count; i++) {
    // Random size between 1 and 4
    const size = Math.random() * 3 + 1; // nosemgrep: nodejs_scan.javascript-crypto-rule-node_insecure_random_generator
    const x = Math.random() * 100; // nosemgrep: nodejs_scan.javascript-crypto-rule-node_insecure_random_generator
    const y = Math.random() * 100; // nosemgrep: nodejs_scan.javascript-crypto-rule-node_insecure_random_generator
    // Random duration between 2 and 5 seconds
    const animationDuration = Math.random() * 3 + 2; // nosemgrep: nodejs_scan.javascript-crypto-rule-node_insecure_random_generator
    stars.push(<Star key={i} size={size} x={x} y={y} animationDuration={animationDuration} />);
  }
  return stars;
};

const LoginPageBackground = React.memo(() => {
  return <SpaceContainer>{generateStars(75)}</SpaceContainer>;
});

export default LoginPageBackground;
