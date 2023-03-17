import Dropdown from './Dropdown';
import {basic, lenses, multi} from './_examples';
import {CSSProperties, useContext, useEffect, useState} from 'react';
import {StateContext} from "../../state";

const styles: { [name: string]: CSSProperties } = {
    dropdown: {
        margin: '0 0',
        minWidth: '180px',
        borderRadius: '20px',
        textAlign: 'center' as const,
    },
}

interface Props {
    style: CSSProperties;
}

const Examples = ({style}: Props) => {
    const {setInput, setLineage} = useContext(StateContext)

    const examples: { [name: string]: { lineage: string; input: string } } = {
        'Basic example': basic,
        'Multiple versions': multi,
        'With lenses': lenses,
    }
    const [example, setExample] = useState<string>(Object.keys(examples)[0]);

    useEffect(() => {
        const {lineage, input} = examples[example];
        setInput(input);
        setLineage(lineage);
    }, [example]) // eslint-disable-line react-hooks/exhaustive-deps


    return (
        <div style={style}>
            <Dropdown id='example' style={styles.dropdown} options={Object.keys(examples)}
                      onChange={(ex: string) => setExample(ex)}/>
        </div>

    )
}

export default Examples;
