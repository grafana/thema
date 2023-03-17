import React from 'react';
import ReactDOM from 'react-dom/client';
import './index.css';
import App from './components/App';
import reportWebVitals from './reportWebVitals';
import {ThemeProvider} from './theme';
import {StateProvider} from './state';

const root = ReactDOM.createRoot(
    document.getElementById('root') as HTMLElement
);

root.render(
    <React.StrictMode>
        <ThemeProvider>
            <StateProvider>
                <App/>
            </StateProvider>
        </ThemeProvider>
    </React.StrictMode>
);

// Now load and run the Go code which will register the Wasm API
const go = new Go();

const sourceFile = 'thema.wasm';

// WebAssembly.instantiateStreaming() is preferred, but not all browsers support it
if (typeof WebAssembly.instantiateStreaming === 'function') {
    WebAssembly.instantiateStreaming(fetch(sourceFile), go.importObject).then(result => {
        go.run(result.instance);
    });
} else {
    fetch(sourceFile).then(response =>
        response.arrayBuffer()
    ).then(bytes =>
        WebAssembly.instantiate(bytes, go.importObject)
    ).then(result => {
        go.run(result.instance);
    });
}

// If you want to start measuring performance in your app, pass a function
// to log results (for example: reportWebVitals(console.log))
// or send to an analytics endpoint. Learn more: https://bit.ly/CRA-vitals
reportWebVitals();
