import { useEffect, useRef } from 'react';
import { useSearchParams } from 'react-router-dom';

// Runs `action` once with a query param's value when present, then strips the param from the URL.
// Reusable for any "confirm on load" flow driven by a link's query string, not just one feature.
export default function useQueryParamAction(param: string, action: (value: string) => void) {
    const [searchParams, setSearchParams] = useSearchParams();
    const firedRef = useRef(false);
    const value = searchParams.get(param);

    useEffect(() => {
        if (value === null || firedRef.current) {
            return;
        }

        firedRef.current = true;
        action(value);

        const next = new URLSearchParams(searchParams);
        next.delete(param);
        setSearchParams(next, { replace: true });
    }, [param, value, action, searchParams, setSearchParams]);
}
