import React, { useState, useEffect } from 'react';
import { useHistory } from 'react-router-dom';
import {withAuthenticationRequired} from "@auth0/auth0-react";
import Loading from "../components/Loading";
import {useApi} from "../api/useApi";
import GameEditForm from '../components/GameEditForm';
import {useRecoilState} from "recoil";
import {gamesState} from "../api/atoms";
import {GamesUri} from "../utils/urls";
import Content from "../components/Content";


export const GameEditComponent = ({match}) => {
    const [, setGames] = useRecoilState(gamesState);
    const [game, setGame] = useState(null);
    const history = useHistory();
    const api = useApi();

    const gameId = match.params.id;

    useEffect(() => {
        if (game == null || gameId !== game.id) {
            api.callApi(`/game/${gameId}`)
                .then(game => setGame(game));
        }
    }, [gameId]);

    const handleSave = updatedGameData => {
        setGame(null);
        api.callApi(`/game/${gameId}`, updatedGameData)
            .then(game => {
                api.callApi("/games").then(games => setGames(games));
                history.push(GamesUri());
                // setGame(game);
            });
    };

    const handleCancel = () => {
        history.push(GamesUri()); // Redirect to the /games route
    };

    if (!game) return <Loading />;

    return (
        <Content>
            <h1>Edit Game #{gameId}</h1>
            <GameEditForm
                initialGame={game}
                onSave={handleSave}
                onCancel={handleCancel}
            />
        </Content>
    );
};

export default withAuthenticationRequired(GameEditComponent, {
    onRedirecting: () => <Loading/>,
});
