import React, { useState, useEffect } from 'react';
import { useHistory } from 'react-router-dom';
import Loading from "../components/Loading";
import { withMockAwareAuth } from "../utils/withMockAwareAuth";
import { useApi } from "../api/useApi";
import GameEditForm from '../components/GameEditForm';
import { useRecoilState } from "recoil";
import { gamesState } from "../api/atoms";
import { GamesUri } from "../utils/urls";
import Content from "../components/Content";
import { Menu } from "../components/Menu";
import { GamesButton } from "../components/GamesButton";


export const GameEditComponent = ({ match }) => {
    const [, setGames] = useRecoilState(gamesState);
    const [game, setGame] = useState(null);
    const history = useHistory();
    const api = useApi();


    const gameId = match.params.id;

    useEffect(() => {
        if (game == null || gameId !== game.id) {
            api.callApi(`/games/${gameId}`)
                .then(game => setGame(game));
        }
    }, [gameId]);

    const handleSave = updatedGameData => {
        setGame(null);
        api.callApi(`/games/${gameId}`, updatedGameData)
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
            <Menu title={`Edit Game #${gameId}`}>
                <GamesButton />
            </Menu>
            <GameEditForm
                initialGame={game}
                onSave={handleSave}
                onCancel={handleCancel}
            />
        </Content>
    );
};

export default withMockAwareAuth(GameEditComponent, {
    onRedirecting: () => <Loading />,
});
