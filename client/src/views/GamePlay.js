import React, {useState, useEffect} from 'react';
import GamePlayer from '../components/GamePlayer'; // Assuming this is your game component

import {useApi} from "../api/useApi";
import Loading from "../components/Loading";
import {useRecoilState} from "recoil";
import {errorsState} from "../api/atoms";


const GamePlay = ({match}) => {
    const [game, setGame] = useState(null);
    const [sessionHash, setSessionHash] = useState(null);
    const api = useApi();

    const gameHash = match.params.hash;

    useEffect(() => {
        if (game == null || gameHash !== game.id) {
            api.callApi(`/public/game/${gameHash}`, null, null)
                .then(game => setGame(game));
        }
    }, [gameHash]);

    useEffect(() => {
        if (game) {
            api.callApi(`/public/session/new`, {gameId: game.id})
                .then(session => {
                    if (session.type === "error") {
                        return;
                    }
                    setSessionHash(session.hash);
                });
        }
    }, [game]);

    return (!game || !sessionHash) ? <Loading /> :
        <GamePlayer game={game} sessionHash={sessionHash} publicSession={true} debug={true} />;

};

export default GamePlay;