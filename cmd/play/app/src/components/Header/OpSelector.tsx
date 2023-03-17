import {TranslateToLatest, TranslateToVersion, ValidateAny, ValidateVersion, Versions} from '../../services/wasm';
import React, {CSSProperties, useContext, useEffect, useState} from 'react';
import {useDebounce} from '../../hooks';
import Dropdown from './Dropdown';
import {StateContext} from '../../state';

const styles: { [name: string]: CSSProperties } = {
    opSelector: {
        gap: '10px',
        display: 'flex',
    },
    dropdown: {
        margin: '0 0',
        minWidth: '60px',
        borderRadius: '20px',
        textAlign: 'center' as const,
    },
    play: {
        margin: '5px 0',
        minWidth: '50px',
        color: '#3d71d9',
        cursor: 'pointer',
    },
}

const OpSelector = () => {
    const {lineage, input} = useContext(StateContext)

    const [version, setVersion] = useState<string>('');

    const ops: { [name: string]: () => void } = {
        ValidateAny: () => ValidateAny(lineage, input),
        ValidateVersion: () => ValidateVersion(lineage, input, version),
        TranslateToLatest: () => TranslateToLatest(lineage, input),
        TranslateToVersion: () => TranslateToVersion(lineage, input, version),
    };

    const [versions, setVersions] = useState<string[]>([]);
    const [operation, setOperation] = useState<string>(Object.keys(ops)[0]);

    const debouncedLineage: string = useDebounce<string>(lineage, 500);

    useEffect((): void => {
        const versions: string[] = Versions(debouncedLineage);

        setVersions(versions);
        setVersion(versions.length > 0 ? versions[0] : '');
    }, [debouncedLineage])

    const versionDropDisabled =
        versions.length === 0 || operation === 'ValidateAny' || operation === 'TranslateToLatest';

    const play = () => ops[operation]();

    return (
        <div style={styles.opSelector}>
            <Dropdown id='operation' style={styles.dropdown} options={Object.keys(ops)}
                      onChange={(op: string) => setOperation(op)}/>
            <Dropdown id='version' style={styles.dropdown} disabled={versionDropDisabled} options={versions}
                      onChange={(ver: string) => setVersion(ver)}/>
            <div style={styles.play} onClick={play}><IconPlay/></div>
        </div>

    )
}

export default OpSelector;

const IconPlay = () =>
    <svg
        viewBox="0 0 24 24"
        fill="currentColor"
        height="1.8em"
        width="3em"
    >
        <path
            d="M12 2C6.486 2 2 6.486 2 12s4.486 10 10 10 10-4.486 10-10S17.514 2 12 2zm0 18c-4.411 0-8-3.589-8-8s3.589-8 8-8 8 3.589 8 8-3.589 8-8 8z"/>
        <path d="M9 17l8-5-8-5z"/>
    </svg>;
