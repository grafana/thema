import {publish} from "../services/terminal";

export const tryOrReport = <T extends Function>(fn: T, ok?: boolean) => {
    try {
        const res = fn();
        if (ok) publish({stdout: 'OK'});
        return res;
    } catch (e: unknown) {
        if (e instanceof Error || e instanceof SyntaxError) {
            publish({stderr: e.message});
        } else {
            publish({stderr: `Unexpected error, look at browser's console for details`});
            console.log(e);
        }
    }
};
