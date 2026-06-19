import IconButton from '@mui/material/IconButton';
import moment from 'moment';
import { useContext, useEffect, useState } from 'react';
import Gravatar from '../common/Gravatar';
import { UserContext } from '../UserContext';

interface Props {
    onClick: (event: React.MouseEvent<HTMLButtonElement>) => void;
}

const NORMAL_GLOW = '251, 191, 36';
const EXPIRING_GLOW = '248, 113, 113';

function AccountMenuGravatar({ onClick }: Props) {
    const user = useContext(UserContext);
    const [expiringSoon, setExpiringSoon] = useState(false);

    useEffect(() => {
        const expiration = user.adminModeExpiration;
        if (!user.adminModeEnabled || !expiration) {
            setExpiringSoon(false);
            return;
        }
        const check = () => setExpiringSoon(moment(expiration).diff(moment()) < 60_000);
        check();
        const interval = setInterval(check, 1000);
        return () => clearInterval(interval);
    }, [user.adminModeEnabled, user.adminModeExpiration]);

    // Steady glow during normal admin mode; flash only when expiring soon to avoid being distracting.
    const glowSx = !user.adminModeEnabled ? {} : expiringSoon ? {
        borderRadius: '50%',
        padding: 0,
        '@keyframes adminGlow': {
            '0%, 100%': { boxShadow: `0 0 0 3px rgba(${EXPIRING_GLOW}, 0.9), 0 0 6px 2px rgba(${EXPIRING_GLOW}, 0.4)` },
            '50%': { boxShadow: `0 0 0 3px rgba(${EXPIRING_GLOW}, 0.9), 0 0 16px 8px rgba(${EXPIRING_GLOW}, 0.6)` },
        },
        animation: 'adminGlow 2s ease-in-out infinite',
    } : {
        borderRadius: '50%',
        padding: 0,
        boxShadow: `0 0 0 3px rgba(${NORMAL_GLOW}, 0.9), 0 0 12px 4px rgba(${NORMAL_GLOW}, 0.45)`,
    };

    return (
        <IconButton onClick={onClick} sx={glowSx}>
            <Gravatar width={32} height={32} email={user.email} />
        </IconButton>
    );
}

export default AccountMenuGravatar;
