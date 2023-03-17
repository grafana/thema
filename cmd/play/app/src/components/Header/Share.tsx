import React, {CSSProperties, useContext, useEffect, useRef, useState} from 'react';
import {fetchState, storeState} from '../../services/store';
import {StateContext} from "../../state";
import {publish} from "../../services/terminal";

const styles: { [name: string]: CSSProperties } = {
    btn: {
        margin: '1px 0',
        minWidth: '100px',
        fontWeight: 700,
        border: 0,
        color: 'white',
        backgroundColor: '#3d71d9',
    },
    input: {
        width: '250',
        fontSize: '15px',
        margin: '1px 0',
    },
}

const hashId = () => window.location.hash.slice(1);

const Share = () => {
    const {input, lineage, setInput, setLineage} = useContext(StateContext)
    const [shareId, setShareId] = useState<string | undefined>(undefined);

    useEffect(() => {
        if (hashId() === '') return

        fetchState(hashId())
            .then(({input, lineage}) => {
                setInput((input || ''));
                setLineage((lineage || ''));
                setShareId((hashId));
            })
            .catch((err: string) => publish({stderr: err}));
    }, []);

    const shareRef = useRef<HTMLInputElement>(null);

    const focusFn = () => {
        shareRef.current?.focus();
        shareRef.current?.setSelectionRange(0, shareRef.current.value.length)
    }

    const shareUrl = (share: string) => (share !== '')
        ? window.location.href.split('#')[0].concat(`#${share}`)
        : '';

    const shareFn = () => {
        storeState({input, lineage})
            .then((id) => {
                setShareId(id);
                setTimeout(focusFn, 500);
                navigator.clipboard.writeText(shareUrl(id));
                window.location.href = shareUrl(id);
            })
    }

    return (
        <>
            <button style={styles.btn} onClick={shareFn}>Share</button>
            <input style={styles.input} ref={shareRef} onClick={focusFn} readOnly={true}
                   value={shareUrl(shareId || '')}/>
        </>
    )
}

export default Share;
