import React, { useState, useEffect } from 'react';
import {withAuthenticationRequired} from "@auth0/auth0-react";
import Loading from "../components/Loading";
import { Container, Row, Col } from 'reactstrap';
import GameViewComponent from '../components/GameView'; // Assuming this is your game component
import DebugLogsComponent from '../components/DebugLogs'; // Assuming this is your debug logs component

import {useApi} from "../api/useApi";


export const GameDebugComponent = ({match}) => {
    const [game, setGame] = useState(null);
    const api = useApi();

    const gameId = match.params.id;

    useEffect(() => {
        if (game == null || gameId !== game.id) {
            api.callApi(`/game/${gameId}`)
                .then(game => setGame(game));
        }
    }, [gameId]);

    const debugLogs = [
        {request: "request1", response: "response1"},
        {request: "request2", response: "response2"},
        {request: "request3", response: "response3"},
        {request: "request4", response: "response4"},
        {request: "request5", response: "response5"},

    ];

    if (!game) return <Loading />;

    return (
        <Row className="flex-grow-1" >
            <Col md={8} className="d-flex flex-column">
                <h1>Debug Game #{gameId}</h1>
                <div className="game-view-container flex-grow-1">
                    <GameViewComponent gameId={gameId} />
                </div>
            </Col>
            <Col md={4} className="d-flex flex-column">
                <div className="debug-logs-container flex-grow-1">
                    <DebugLogsComponent logs={debugLogs} />
                </div>
            </Col>
        </Row>
    );
};

export default withAuthenticationRequired(GameDebugComponent, {
    onRedirecting: () => <Loading/>,
});