// Vitest setup file
import '@testing-library/jest-dom';

// No need to manually extend matchers - the import above does it automatically
// when using with Vitest

import { afterEach } from 'vitest';
import { cleanup } from '@testing-library/react';

// Clean up after each test
afterEach(() => {
  cleanup();
});
