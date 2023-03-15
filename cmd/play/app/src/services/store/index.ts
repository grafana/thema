import {defaultState, State} from '../../components/state';

const SEPARATOR = "*+*+*ThemaPlaygroundSeparator*+*+*"
const GO_PLAYGROUND_API = 'https://play.golang.org';

export const storeState = async (state: State): Promise<string> => {
    const snippet = state.lineage.concat(SEPARATOR, state.input)
    const response = await window.fetch(`${GO_PLAYGROUND_API}/share`, {
        method: 'POST',
        body: snippet,
    })

    if (response.ok) {
        return Promise.resolve(response.text());
    }
    return Promise.reject(response.text())
}

export const fetchState = async (id: string): Promise<State> => {
    const response = await window.fetch(`${GO_PLAYGROUND_API}/p/${id}.go`, {
        method: 'GET',
    })

    if (response.ok) {
        const [lineage, input] = await response.text().then((res: string) => res.split(SEPARATOR));
        return Promise.resolve({...defaultState, lineage, input, shareId: id});
    }
    return Promise.reject(response.text())
}
