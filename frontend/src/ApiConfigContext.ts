import { createContext } from 'react';

export interface ApiConfig {
    tharsisSupportUrl: string;
    serviceDiscoveryHost: string;
}

// ApiConfig will never be null so it's safe to use an empty object as the default value here
export const ApiConfigContext = createContext<ApiConfig>({} as ApiConfig);
