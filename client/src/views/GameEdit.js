import React, { useState, useEffect } from 'react';
import { useHistory } from 'react-router-dom';
import {withAuthenticationRequired} from "@auth0/auth0-react";
import Loading from "../components/Loading";
import {useApi} from "../api/useApi";
import GameEditForm from '../components/GameEditForm';


export const GameEditComponent = ({match}) => {
    const [game, setGame] = useState(null);
    const history = useHistory();
    const api = useApi();

    const gameId = match.params.id;

    useEffect(() => {
        if (game == null || gameId !== game.ID) {
            api.callApi(`/game/${gameId}`)
                .then(game => setGame(game));
        }
    }, [gameId]);

    const handleSave = updatedGameData => {
        setGame(null);
        api.callApi(`/game/${gameId}`, updatedGameData)
            .then(game => setGame(game));
    };

    const handleCancel = () => {
        history.push('/games'); // Redirect to the /games route
    };

    if (!game) return <Loading />;

    return (
        <>
            <div className="mb-5">
                <h1>Edit Game #{gameId}</h1>
                <GameEditForm
                    initialGame={game}
                    onSave={handleSave}
                    onCancel={handleCancel}
                />
            </div>
        </>
    );
};

export default withAuthenticationRequired(GameEditComponent, {
    onRedirecting: () => <Loading/>,
});
