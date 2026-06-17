import { createContext } from 'react';

export interface User {
    id: string;
    username: string;
    email: string;
    admin: boolean;
    adminModeEnabled: boolean;
    adminModeExpiration: string | null;
}

// User will never be null so it's safe to use an empty object as the default value here
export const UserContext = createContext<User>({} as User);
