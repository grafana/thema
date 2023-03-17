interface TerminalUpdate {
    stdout?: string;
    stderr?: string;
}

export const publish = (update: TerminalUpdate) =>
    window.postMessage(update, '*');


export const subscribe = (handler: (update: TerminalUpdate) => void) => {
    const messageEventHandler = (event: MessageEvent<TerminalUpdate>) =>
        event.data !== undefined && handler(event.data);

    const attachEventListener = () => {
        window.addEventListener('message', messageEventHandler);
    };
    const detachEventListener = () => {
        window.removeEventListener('message', messageEventHandler);
    };

    attachEventListener();
    return {unsubscribe: detachEventListener};
};
