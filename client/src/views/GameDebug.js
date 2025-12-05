import React, { useState, useEffect } from 'react';
import Loading from "../components/Loading";
import {withMockAwareAuth} from "../utils/withMockAwareAuth";
import GamePlayer from '../components/GamePlayer'; // Assuming this is your game component

import {useApi} from "../api/useApi";

export const GameDebugComponent = ({match}) => {
    const [game, setGame] = useState(null);
    const [sessionHash, setSessionHash] = useState(null);
    const api = useApi();

    const gameId = match.params.id;

    useEffect(() => {
        if (game == null || gameId !== game.id) {
            api.callApi(`/game/${gameId}`)
                .then(game => setGame(game));
        }
    }, [gameId]);

    useEffect(() => {
        if (game) {
            api.callApi(`/session/new`, {gameId: game.id})
                .then(session => {
                    console.log("new session: ", session);
                    setSessionHash(session.hash);
                });
        }
    }, [game]);


    return (
        (!game || !sessionHash) ? <Loading />
        : <GamePlayer game={game} sessionHash={sessionHash} debug={true} />
    );

};

export default withMockAwareAuth(GameDebugComponent, {
    onRedirecting: () => <Loading/>,
});
