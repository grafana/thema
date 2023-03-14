import React, {useRef} from "react";
import * as monaco from "monaco-editor";
import Editor, {Monaco} from "@monaco-editor/react";

interface Props {
    value: string;
    onChange: (value: string) => void;
    isReadOnly?: boolean;
}

export const CodeEditor = ({value, onChange, isReadOnly}: Props) => {
    // Monaco / Editor
    const editorRef = useRef<null | monaco.editor.IStandaloneCodeEditor>(null);
    const monacoRef = useRef<null | Monaco>(null);

    const handleEditorDidMount = (editor: monaco.editor.IStandaloneCodeEditor, monaco: Monaco) => {
        editorRef.current = editor;
        monacoRef.current = monaco;
    }

    const readOnlyProps: monaco.editor.IStandaloneEditorConstructionOptions = {
        lineNumbers: 'off',
        minimap: {enabled: false},
    }

    const opts = isReadOnly ? readOnlyProps : {}


    return (
        <Editor
            options={opts}
            height="100%"
            value={value}
            defaultValue="// some comment"
            onMount={handleEditorDidMount}
            onChange={(val?: string) => onChange(val || '')}
        />
    );
}
