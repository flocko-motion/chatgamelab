import { atom } from 'recoil';

const gamesStateExample = [
    { name: "Game 1", shareState: "public", editLink: "/edit/game1", ownerName: "Alice" },
    { name: "Game 2", shareState: "private", editLink: "/edit/game2", ownerName: "Bob" },
    // ... more games
];

export const userState = atom({
    key: 'user',
    default: null,
});

export const gamesState = atom({
    key: 'games',
    default: gamesStateExample,
});

export const loadingState = atom({
    key: 'loading',
    default: false,
})

export const errorsState = atom({
    key: 'errors',
    default: [],
})

export const mockAuthState = atom({
    key: 'mockAuth',
    default: false,
})

// CGL JWT token auth (for dev login via CLI)
const hasCglToken = !!localStorage.getItem('cgl_token');
console.log('atoms.js: cgl_token check:', hasCglToken, localStorage.getItem('cgl_token')?.substring(0, 20));
export const cglAuthState = atom({
    key: 'cglAuth',
    default: hasCglToken,
})