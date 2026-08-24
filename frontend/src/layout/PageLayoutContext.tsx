import Box from '@mui/material/Box';
import { createContext, ReactNode, useCallback, useContext, useLayoutEffect, useState } from 'react';

export type PageLayoutSize = 'normal' | 'wide' | 'fullscreen';

const SIZE_MAP: Record<PageLayoutSize, number | undefined> = {
    normal: 1200,
    wide: 1400,
    fullscreen: undefined,
};

const PageLayoutContext = createContext<(size: PageLayoutSize | undefined) => void>(() => {});

interface PageLayoutProviderProps {
    children: ReactNode;
    size?: PageLayoutSize;
}

export function PageLayoutProvider({ children, size }: PageLayoutProviderProps) {
    const [sizeOverride, setSizeOverride] = useState<PageLayoutSize | undefined>(size);
    const setSize = useCallback((s: PageLayoutSize | undefined) => setSizeOverride(s), []);
    const currentSize = sizeOverride ?? 'normal';
    const maxWidth = SIZE_MAP[currentSize];
    const mx = currentSize === 'fullscreen' ? 2 : 'auto';
    return (
        <PageLayoutContext.Provider value={setSize}>
            <Box maxWidth={maxWidth} mx={mx} padding={2}>
                {children}
            </Box>
        </PageLayoutContext.Provider>
    );
}

export function usePageLayout(size: PageLayoutSize) {
    const setSize = useContext(PageLayoutContext);
    useLayoutEffect(() => {
        setSize(size);
        return () => setSize(undefined);
    }, [size, setSize]);
}
