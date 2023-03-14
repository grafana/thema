import React, {useState} from 'react';
import './App.css';
import {CodeEditor} from './components/CodeEditor';
import {Button} from 'react-bootstrap';

const App = () => {
    const [lineage, setLineage] = useState<string>("")
    const [data, setData] = useState<string>("")
    const [output, setOutput] = useState<string>("")

    const validateAnyFn = () => {
        // @ts-ignore
        const res = validateAny(lineage, data);
        if (res.error !== '') {
            setOutput(`'ValidateAny' failed: ${res.error}`)
        } else {
            setOutput(`Input data matches schema version: ${res.result}`)
        }
    }

    const translateToLatestFn = () => {
        // @ts-ignore
        const res = translateToLatest(lineage, data);
        if (res.error !== '') {
            setOutput(`'TranslateToLatest' failed: ${res.error}`)
            return
        }

        const d = JSON.parse(res.result)
        setOutput(`From: ${d.from}
To: ${d.to}
        
Result:        
${JSON.stringify(d.result, null, "\t")}
        
Lacunas:  
${JSON.stringify(d.lacunas, null, "\t")}
`)
    }

    return (
        <div className="App" style={{display: 'flex', flexWrap: 'wrap'}}>
            <div style={{
                display: 'flex',
                gap: '20px',
                height: '80px',
                width: '100vw',
                background: 'green',
                padding: '10px 40px'
            }}>
                <h1 style={{color: 'white', marginRight: '5vw'}}>Thema Playground</h1>
                <Button onClick={validateAnyFn} variant="warning">ValidateAny</Button>{' '}
                <Button onClick={translateToLatestFn} variant="warning">TranslateToLatest</Button>{' '}
                <Button variant="warning">Primary</Button>{' '}
            </div>


            <div style={{height: '80vh', width: '33vw', paddingTop: '20px'}}>
                <h4 style={{textAlign: 'left', paddingLeft: '1vw'}}>Lineage:</h4>
                <div style={{height: '80vh', margin: '1vh 1vw', border: '1px dashed green'}}>
                    <CodeEditor value={lineage} onChange={setLineage}/>
                </div>
            </div>

            <div style={{height: '80vh', width: '33vw', paddingTop: '20px'}}>
                <h4 style={{textAlign: 'left', paddingLeft: '1vw'}}>Input data (JSON):</h4>
                <div style={{height: '80vh', margin: '1vh 1vw', border: '1px dashed green'}}>
                    <CodeEditor value={data} onChange={setData}/>
                </div>
            </div>

            <div style={{height: '80vh', width: '33vw', paddingTop: '20px'}}>
                <h4 style={{textAlign: 'left', paddingLeft: '1vw'}}>Output:</h4>
                <div style={{height: '80vh', margin: '1vh 1vw', border: '1px dashed darkblue'}}>
                    <CodeEditor isReadOnly={true} value={output} onChange={() => {}}/>
                </div>
            </div>
        </div>
    );
}

export default App;
