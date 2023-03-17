import React, {useContext} from 'react';
import './App.css';
import CodeEditor from './CodeEditor';
import Header from './Header';
import Column from './Column';
import Console from './Console';
import {ThemeContext} from '../theme';
import {StateContext} from "../state";

const App = () => {
    const {theme} = useContext(ThemeContext);
    const {input, lineage, setInput, setLineage} = useContext(StateContext)

    return (
        <div className={`App theme-${theme}`} style={{display: 'flex', flexWrap: 'wrap', alignContent: 'flex-start'}}>
            <Header/>
            <Column title='LINEAGE (CUE)' color='green'>
                <CodeEditor value={lineage} language='go'
                            onChange={(lineage?: string) => setLineage(lineage || '')}/>
            </Column>
            <Column title='INPUT DATA (JSON)' color='green'>
                <CodeEditor value={input} language='json'
                            onChange={(input?: string) => setInput(input || '')}/>
            </Column>
            <Column title='OUTPUT' color='darkblue'>
                <Console/>
            </Column>

        </div>
    );
}

export default App;
