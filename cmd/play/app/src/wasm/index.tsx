import React, { useEffect } from 'react';

async function loadWasm(): Promise<void> {
    const go = new Go();

    WebAssembly.instantiateStreaming(fetch('thema.wasm'), go.importObject).then(result => {
        go.run(result.instance);
    });
}

export const LoadWasm: React.FC<React.PropsWithChildren<{}>> = (props) => {
    const [isLoading, setIsLoading] = React.useState(true);

    useEffect(() => {
        loadWasm().then(() => {
            setIsLoading(false);
        });
    }, []);

    if (isLoading) {
        return (
            <div>
                loading WebAssembly...
            </div>
        );
    } else {
        return <React.Fragment>{props.children}</React.Fragment>;
    }
};
