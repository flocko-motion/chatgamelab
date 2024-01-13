import React, { useState, useEffect } from 'react';
import {withAuthenticationRequired} from "@auth0/auth0-react";
import Loading from "../components/Loading";
import { Row, Col } from 'reactstrap';
import GamePlayer from '../components/GamePlayer'; // Assuming this is your game component
import DebugLogsComponent from '../components/DebugLogs'; // Assuming this is your debug logs component

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

    const debugLogs = [
        {request: "request1", response: "response1"},
        {request: "request2", response: "response2"},
        {request: "request3", response: "response3"},
        {request: "request4", response: "response4"},
        {request: "request5", response: "response5"},

    ];

    if (!game || !sessionHash) return <Loading />;

    return (
        <Row className="flex-grow-1 h-100">
            <Col md={8} className="d-flex flex-column h-100 p-0">
                <GamePlayer game={game} sessionHash={sessionHash} />
            </Col>
            <Col md={4} className="d-flex flex-column h-100 p-0">
                <DebugLogsComponent logs={debugLogs} />
            </Col>
        </Row>
    );
};

export default withAuthenticationRequired(GameDebugComponent, {
    onRedirecting: () => <Loading/>,
});
