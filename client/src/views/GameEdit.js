import React, { useState, useEffect } from 'react';
import {withAuthenticationRequired} from "@auth0/auth0-react";
import Loading from "../components/Loading";
import {useApi} from "../api/useApi";



export const GameEditComponent = ({match}) => {
    const [game, setGame] = useState(null);
    const gameId = match.params.id;

    const api = useApi();

    useEffect(() => {
        if (game == null || gameId !== game.ID) {

            // Call your API here
            console.log(`Game ID changed to: ${gameId}`);

            api.callApi(`/game/${gameId}`).then(game => setGame(game));
        }
    }, [gameId]);

    return (
        <>
            <div className="mb-5">
                <h1>Edit Game #{gameId}</h1>
            </div>
        </>
    );
};

export default withAuthenticationRequired(GameEditComponent, {
    onRedirecting: () => <Loading/>,
});
