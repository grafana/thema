import React, {useReducer} from 'react';

export interface State {
    input: string;
    setInput: (input: string) => void;

    lineage: string;
    setLineage: (lineage: string) => void;
}

const initialState: State = {
    input: '',
    setInput: (_: string): void => {
    },

    lineage: '',
    setLineage: (_: string): void => {
    },
};

enum StateActionKind {
    INPUT = 'INPUT',
    LINEAGE = 'LINEAGE',
}

interface StateAction {
    type: StateActionKind;
    payload: string;
}

const stateReducer = (state: Partial<State>, {type, payload}: StateAction): Partial<State> => {
    if (type === StateActionKind.INPUT) {
        return {...state, input: payload};
    }

    if (type === StateActionKind.LINEAGE) {
        return {...state, lineage: payload};
    }

    return state;
};

export const StateContext: React.Context<State> = React.createContext(initialState);

export const StateProvider = (props: React.PropsWithChildren) => {
    const [{input, lineage}, dispatch] = useReducer(stateReducer, initialState);

    const setInput = (input: string) => dispatch({type: StateActionKind.INPUT, payload: input});
    const setLineage = (lineage: string) => dispatch({type: StateActionKind.LINEAGE, payload: lineage});

    const state: State = {input: (input || ''), lineage: (lineage || ''), setInput, setLineage};

    return <StateContext.Provider value={state}>{props.children}</StateContext.Provider>;
}
