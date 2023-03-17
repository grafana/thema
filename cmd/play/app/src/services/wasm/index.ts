import {publish} from '../terminal';

export const Format = (lineage: string): string => {
    // @ts-ignore
    const res = format(lineage);
    if (res.error !== '') {
        throw new Error(res.error);
    }
    return res.result;
}

export const Versions = (lineage: string): string[] => {
    // This function is called "automatically", on every
    // lineage change (debounced), so we need to prevent
    // it to be executed before WASM API is loaded.

    // @ts-ignore
    if (typeof getLineageVersions !== 'function') {
        return [];
    }

    // @ts-ignore
    const res = getLineageVersions(lineage);
    if (res.error !== '') {
        publish({stderr: `Lineage: ${res.error}`});
        return [];
    }
    publish({stderr: ''});
    return JSON.parse(res.result);
}

export const ValidateAny = (lineage: string, input: string): void => {
    // @ts-ignore
    const res = validateAny(lineage, input);
    if (res.error !== '') {
        publish({stderr: `'ValidateAny' failed: ${res.error}`});
        return
    }
    publish({stdout: `Input data matches schema version: ${res.result}`});
}

export const ValidateVersion = (lineage: string, input: string, version: string): void => {
    // @ts-ignore
    const res = validateVersion(lineage, input, version);
    if (res.error !== '') {
        publish({stderr: `'ValidateVersion' failed: ${res.error}`});
        return
    }
    publish({stdout: `Input data validated successfully: ${res.result}`});
}

export const TranslateToLatest = (lineage: string, input: string): void => {
    // @ts-ignore
    const res = translateToLatest(lineage, input);
    if (res.error !== '') {
        publish({stderr: `'TranslateToLatest' failed: ${res.error}`});
        return
    }
    publish({stdout: translateResultToString(res.result)});
}

export const TranslateToVersion = (lineage: string, input: string, version: string): void => {
    // @ts-ignore
    const res = translateToLatest(lineage, input);
    if (res.error !== '') {
        publish({stderr: `'TranslateToVersion' failed: ${res.error}`});
        return
    }
    publish({stdout: translateResultToString(res.result)});
}

const translateResultToString = (res: string): string => {
    const data = JSON.parse(res)

    return `From: ${data.from}
To: ${data.to}
        
Result:        
${JSON.stringify(data.result, null, "\t")}
        
Lacunas:  
${JSON.stringify(data.lacunas, null, "\t")}`;
}
