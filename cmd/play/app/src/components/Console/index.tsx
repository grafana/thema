import React, {useContext, useEffect, useState} from 'react';
import Terminal, {ColorMode, TerminalInput, TerminalOutput} from 'react-terminal-ui';
import {subscribe} from '../../services/terminal';
import {Theme, ThemeContext} from '../../theme';
import {nanoid} from "nanoid";

const DEFAULT_TEXT = "Select a Thema operation and click play";
const DEFAULT_INPUT = <TerminalInput key={nanoid()}>{`${DEFAULT_TEXT}...`}</TerminalInput>;

interface ConsoleInput {
    stdout: TerminalInput[];
    stderr: TerminalInput[];
}

const defaultConsoleInput = (): ConsoleInput => {
    return {stdout: [], stderr: []}
};

const Console = () => {
    const {theme} = useContext(ThemeContext);
    const color = (theme === Theme.light) ? ColorMode.Light : ColorMode.Dark;
    const [input, setInput] = useState<ConsoleInput>(defaultConsoleInput());

    useEffect(() => {
        subscribe(({stderr, stdout}) => {
            const inp: ConsoleInput = defaultConsoleInput();
            if (stderr) inp.stderr.push(<TerminalInput key={nanoid()}>{stderr}</TerminalInput>);
            if (stdout) inp.stdout.push(<TerminalInput key={nanoid()}>{stdout}</TerminalInput>);
            setInput(inp);
        })
    });

    return <Terminal name='Console' colorMode={color}>
        <TerminalOutput>~~~~~~ Standard Error ~~~~~~ </TerminalOutput>
        {input.stderr.length > 0 ? input.stderr : <TerminalOutput/>}
        <TerminalOutput>~~~~~~ Standard Output ~~~~~~ </TerminalOutput>
        {input.stdout.length > 0 ? input.stdout : <TerminalOutput/>}
    </Terminal>
};

export default Console;
