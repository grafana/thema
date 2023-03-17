import React, {useReducer} from 'react';

export enum Theme {
    dark = 'dark',
    light = 'light',
}

interface ThemeCfg {
    theme: Theme
    toggleTheme: React.DispatchWithoutAction
}

const initialState: ThemeCfg = {
    theme: Theme.dark,
    toggleTheme: (): void => {
    },
};

const themeReducer = (state: ThemeCfg): ThemeCfg => {
    const theme: Theme = (state.theme === Theme.dark)
        ? Theme.light
        : Theme.dark;

    return {...state, theme};
};

export const ThemeContext: React.Context<ThemeCfg> = React.createContext(initialState);

export const ThemeProvider = (props: React.PropsWithChildren) => {
    const [state, dispatch] = useReducer(themeReducer, initialState);
    const themeCfg: ThemeCfg = {theme: state.theme, toggleTheme: dispatch};
    return <ThemeContext.Provider value={themeCfg}>{props.children}</ThemeContext.Provider>;
}
