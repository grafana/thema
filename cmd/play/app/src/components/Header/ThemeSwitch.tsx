import React, {CSSProperties, useContext} from 'react';
import {Theme, ThemeContext} from '../../theme';

const IconMoon = () =>
    <svg
        viewBox="0 0 24 24"
        fill="currentColor"
        height="2.2em"
        width="2em"
    >
        <path strokeLinecap="round" strokeLinejoin="round"
              d="M21.752 15.002A9.718 9.718 0 0118 15.75c-5.385 0-9.75-4.365-9.75-9.75 0-1.33.266-2.597.748-3.752A9.753 9.753 0 003 11.25C3 16.635 7.365 21 12.75 21a9.753 9.753 0 009.002-5.998z"/>
    </svg>;


const IconSun = () =>
    <svg
        viewBox="0 0 24 24"
        fill="currentColor"
        height="2.2em"
        width="2em"
    >
        <path
            d="M6.995 12c0 2.761 2.246 5.007 5.007 5.007s5.007-2.246 5.007-5.007-2.246-5.007-5.007-5.007S6.995 9.239 6.995 12zM11 19h2v3h-2zm0-17h2v3h-2zm-9 9h3v2H2zm17 0h3v2h-3zM5.637 19.778l-1.414-1.414 2.121-2.121 1.414 1.414zM16.242 6.344l2.122-2.122 1.414 1.414-2.122 2.122zM6.344 7.759L4.223 5.637l1.415-1.414 2.12 2.122zm13.434 10.605l-1.414 1.414-2.122-2.122 1.414-1.414z"/>
    </svg>


interface Props {
    style: CSSProperties;
}

const ThemeSwitch = ({style}: Props) => {
        const {theme, toggleTheme} = useContext(ThemeContext);

        return (<div style={style} onClick={toggleTheme}>
                {(theme === Theme.dark) ? <IconSun/> : <IconMoon/>}
            </div>
        )
    }
;

export default ThemeSwitch
