import {Format} from '../wasm';

export const fmtCue = (lineage: string): string =>
    Format(lineage);

export const fmtJson = (input: string): string =>
    JSON.stringify(JSON.parse(input), null, "\t");
