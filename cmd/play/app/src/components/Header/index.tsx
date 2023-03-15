import {Button} from 'react-bootstrap';
import React, {Dispatch, SetStateAction, useEffect, useRef, useState} from 'react';
import {State} from '../state';
import {
    Format,
    TranslateToLatest,
    TranslateToVersion,
    ValidateAny,
    ValidateVersion,
    Versions
} from '../../services/wasm';
import {IconPlay} from './IconPlay';
import Dropdown from './Dropdown';
import {useDebounce} from '../../hooks';
import {storeState} from "../../services/store";

const styles = {
    header: {
        display: 'flex',
        gap: '20px',
        height: '80px',
        width: '100vw',
        background: 'green',
        padding: '10px 40px'
    },
    title: {
        color: 'white',
        marginRight: '4vw',
    },
    opSelector: {
        gap: '10px',
        display: 'flex',
        margin: '5px 0',
    },
    dropdown: {
        margin: '5px 0',
        minWidth: '60px',
        border: '0px solid transparent',
        borderRadius: '20px',
        borderRight: '4px solid transparent',
        textAlign: 'center' as const,
    },
    play: {
        margin: '5px 0',
        minWidth: '50px',
        color: '#ffc107',
        cursor: 'pointer',
    },
    btn: {
        margin: '10px 0',
        minWidth: '100px',
    },
    btnFmt: {
        marginLeft: '3vw',
        marginRight: '10vw',
    },
    input: {
        width: '250',
        fontSize: '15px',
        margin: '7px 0',
    }
};

interface Props {
    state: State;
    setState: Dispatch<SetStateAction<State>>;
}

const Index = ({state, setState}: Props) => {
    const {lineage, input, share} = state;

    const formatFn = () => setState({
        ...state,
        lineage: Format(lineage),
        input: JSON.stringify(JSON.parse(input), null, "\t"),
    });

    const shareRef = useRef<HTMLInputElement>(null);

    const focusFn = () => {
        shareRef.current?.focus();
        shareRef.current?.setSelectionRange(0, shareRef.current.value.length)
    }

    const shareUrl = (share: string) => (share !== '')
        ? window.location.href.split('#')[0].concat(`#${share}`)
        : '';

    const shareFn = () => {
        storeState(state)
            .then((id) => {
                setState({...state, share: id});
                setTimeout(focusFn, 500);
                navigator.clipboard.writeText(shareUrl(id));
                window.location.href = shareUrl(id);
            })
    }

    return (
        <div style={styles.header}>
            <h1 style={styles.title}>Thema Playground</h1>
            <OpSelector state={state} setState={setState}/>
            <Button style={{...styles.btn, ...styles.btnFmt}} onClick={formatFn} variant="warning">Format</Button>
            <Button style={styles.btn} onClick={shareFn} variant="warning">Share</Button>
            <input style={styles.input} ref={shareRef} onClick={focusFn} readOnly={true} value={shareUrl(share)}/>
        </div>
    );
}

export default Index;

const OpSelector = ({state, setState}: Props) => {
    const defaultOp = 'ValidateAny';

    const ops: { [name: string]: () => void } = {
        'ValidateAny': () => {
            setState({...state, output: ValidateAny(state.lineage, state.input)});
        },
        'ValidateVersion': () => {
            setState({...state, output: ValidateVersion(state.lineage, state.input, version)});
        },
        'TranslateToLatest': () => {
            setState({...state, output: TranslateToLatest(state.lineage, state.input)});
        },
        'TranslateToVersion': () => {
            setState({...state, output: TranslateToVersion(state.lineage, state.input, version)});
        },
    }

    const [version, setVersion] = useState<string>('');
    const [versions, setVersions] = useState<string[]>([]);
    const [operation, setOperation] = useState<string>(defaultOp);

    const debouncedLineage: string = useDebounce<string>(state.lineage, 500);

    useEffect(() => {
        setVersions(Versions(debouncedLineage));
    }, [debouncedLineage])

    const versionDropDisabled =
        versions.length === 0 || operation === 'ValidateAny' || operation === 'TranslateToLatest';

    const play = () => ops[operation]();

    return (
        <div style={styles.opSelector}>
            <Dropdown id='operation' style={styles.dropdown} options={Object.keys(ops)}
                      onChange={(op) => setOperation(op)}/>
            <Dropdown id='version' style={styles.dropdown} disabled={versionDropDisabled} options={versions}
                      onChange={(ver) => setVersion(ver)}/>
            <div style={styles.play} onClick={play}><IconPlay/></div>
        </div>

    )
}
