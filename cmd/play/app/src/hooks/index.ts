import {useEffect, useState} from "react";

export function useDebounce<T>(value: T, ms: number): T {
    const [debouncedValue, setDebouncedValue] = useState(value);

    useEffect(
        () => {
            const handler = setTimeout(() => {
                setDebouncedValue(value);
            }, ms);
            return () => {
                clearTimeout(handler);
            };
        },
        [value, ms]
    );

    return debouncedValue;
}
