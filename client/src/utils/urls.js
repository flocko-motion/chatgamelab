export const SharePlayUrl = (hash) => {
    return `${window.location.origin}/play/${hash}`;
}

export const SharePlayUri = (hash) => {
    return `/play/${hash}`;
}

export const DebugUri = (gameId) => {
    return `/debug/${gameId}`;
}

export const EditUri = (gameId) => {
    return `/edit/${gameId}`;
}

export const GamesUri = () => {
    return `/games`;
}
