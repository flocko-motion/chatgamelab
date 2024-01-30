import React, { useState, useEffect } from 'react';
import GamePlayer from '../components/GamePlayer'; // Assuming this is your game component

import {useApi} from "../api/useApi";
import Loading from "../components/Loading";


const  GamePlay = ({match}) => {
    const [game, setGame] = useState(null);
    const api = useApi();

    const gameHash = match.params.hash;

    useEffect(() => {
        if (game == null || gameHash !== game.id) {
            api.callApi(`/game/${gameHash}`)
                .then(game => setGame(game));
        }
    }, [gameHash]);

    return game ? <GamePlayer gameId={gameHash} /> : <Loading />;
};

export default GamePlay;