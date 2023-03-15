import React, {useEffect, useState} from 'react';
import './App.css';
import CodeEditor from './CodeEditor';
import Header from './Header';
import Column from './Column';
import {defaultState, State} from './state';
import {fetchState} from "../services/store";

const App = () => {
    const [state, setState] = useState<State>(defaultState);

    useEffect(() => {
            const id = window.location.hash.slice(1);
            if (id === '') {
                return
            }

            fetchState(id)
                .then((s: State) => setState({...s, share: id}))
                .catch((err: string) => console.log(err));
        },
        [setState]
    );

    return (
        <div className="App" style={{display: 'flex', flexWrap: 'wrap'}}>
            <Header state={state} setState={setState}/>
            <Column title='Lineage' color='green'>
                <CodeEditor value={state.lineage}
                            onChange={(lineage?: string) => setState({...state, lineage: lineage || ''})}/>
            </Column>
            <Column title='Input data (JSON)' color='green'>
                <CodeEditor value={state.input}
                            onChange={(input?: string) => setState({...state, input: input || ''})}/>
            </Column>
            <Column title='Output' color='darkblue'>
                <CodeEditor isReadOnly={true} value={state.output}/>
            </Column>
        </div>
    );
}

export default App;
