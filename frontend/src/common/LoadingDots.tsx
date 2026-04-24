import { useEffect, useState } from 'react';

function LoadingDots() {
    const [active, setActive] = useState(0);

    useEffect(() => {
        const interval = setInterval(() => {
            setActive(a => (a + 1) % 3);
        }, 400);
        return () => clearInterval(interval);
    }, []);

    return (
        <span style={{ fontSize: '36px', letterSpacing: '-2px', lineHeight: '0.5' }}>
            {[0, 1, 2].map(i => (
                <span key={i} style={{
                    opacity: i === active ? 1 : 0.3,
                    transition: 'opacity 0.6s ease-in-out'
                }}>.</span>
            ))}
        </span>
    );
}

export default LoadingDots;
