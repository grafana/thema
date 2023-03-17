import {State} from '../../state'

const SEPARATOR: string = "*+*+*ThemaPlaygroundSeparator*+*+*"
const GO_PLAYGROUND_API: string = 'https://play.golang.org';

export const storeState = async ({input, lineage}: Partial<State>): Promise<string> => {
    const snippet: string = (lineage || '').concat(SEPARATOR, (input || ''));
    const response: Response = await window.fetch(`${GO_PLAYGROUND_API}/share`, {
        method: 'POST',
        body: snippet,
    })

    if (response.ok) {
        return Promise.resolve(response.text());
    }
    return Promise.reject(response.text())
}

export const fetchState = async (id: string): Promise<Partial<State>> => {
    const response: Response = await window.fetch(`${GO_PLAYGROUND_API}/p/${id}.go`, {
        method: 'GET',
    })

    if (response.ok) {
        const [lineage, input] = await response.text().then((res: string) => res.split(SEPARATOR));
        return Promise.resolve({lineage, input});
    }
    return Promise.reject(response.text())
}
