import React, { useState, useEffect } from 'react';
import {withAuthenticationRequired} from "@auth0/auth0-react";
import Loading from "../components/Loading";
import { Row, Col } from 'reactstrap';
import GamePlayer from '../components/GamePlayer'; // Assuming this is your game component
import DebugLogsComponent from '../components/DebugLogs'; // Assuming this is your debug logs component

import {useApi} from "../api/useApi";


export const GamePlayComponent = ({match}) => {
    const [game, setGame] = useState(null);
    const api = useApi();

    const gameId = match.params.hash;

    useEffect(() => {
        if (game == null || gameId !== game.id) {
            api.callApi(`/game/${gameId}`)
                .then(game => setGame(game));
        }
    }, [gameId]);

    // if (!game) return <Loading />;

    return (
        <GamePlayer gameId={gameId} />
    );
};

export default withAuthenticationRequired(GamePlayComponent, {
    onRedirecting: () => <Loading/>,
});
