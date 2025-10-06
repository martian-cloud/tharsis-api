import { createContext, useContext, useState, ReactNode } from 'react';

interface AppHeaderHeightContextType {
    headerHeight: number;
    setHeaderHeight: (height: number) => void;
}

// Create Context
const AppHeaderHeightContext = createContext<AppHeaderHeightContextType>({
    headerHeight: 64,
    setHeaderHeight: () => {
        // Default implementation - no-op when context is not provided
    },
});

// Context Provider Component
export const AppHeaderHeightProvider = ({ children }: { children: ReactNode }) => {
    const [headerHeight, setHeaderHeight] = useState(64);

    return (
        <AppHeaderHeightContext.Provider value={{ headerHeight, setHeaderHeight }}>
            {children}
        </AppHeaderHeightContext.Provider>
    );
};

// Custom hook for easier context consumption
export const useAppHeaderHeight = () => useContext(AppHeaderHeightContext);
