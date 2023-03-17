import React, {CSSProperties, useContext} from 'react';
import OpSelector from './OpSelector';
import {fmtCue, fmtJson} from '../../services/format';
import {tryOrReport} from '../../helpers';
import Examples from './Examples';
import ThemeSwitch from './ThemeSwitch';
import {StateContext} from '../../state';
import Share from './Share';

const styles: { [name: string]: CSSProperties } = {
    header: {
        display: 'flex',
        gap: '20px',
        height: '60px',
        width: '100vw',
        padding: '10px 40px',
        border: 'rgba(204, 204, 220, 0.07) solid 1px',
        borderRadius: '2px',
    },
    title: {
        marginRight: '4vw',
    },
    btn: {
        margin: '1px 0',
        minWidth: '100px',
        fontWeight: 700,
        border: 0,
        color: 'white',
        backgroundColor: '#3d71d9',
    },
    btnFmt: {
        marginLeft: '5vw',
    },
    input: {
        width: '250',
        fontSize: '15px',
        margin: '1px 0',
    },
    examples: {
        marginLeft: '10vw',
        marginRight: '1vw',
        display: 'flex',
    },
    themeSwitch: {
        color: '#3d71d9',
        cursor: 'pointer',
    }
};

const Header = () => {
    const {input, lineage, setInput, setLineage} = useContext(StateContext)

    const formatFn = () => {
        tryOrReport(() => {
            setInput(fmtJson(input));
            setLineage(fmtCue(lineage));
        }, true);
    }

    return (
        <div className='header' style={styles.header}>
            <h3 style={styles.title}>Thema Playground</h3>
            <OpSelector/>
            <button style={{...styles.btn, ...styles.btnFmt}} onClick={formatFn}>Format</button>
            <Share/>
            <Examples style={styles.examples}/>
            <ThemeSwitch style={styles.themeSwitch}/>
        </div>
    );
}

export default Header;
