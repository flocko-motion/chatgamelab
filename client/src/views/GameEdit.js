import React from 'react';
import { useRecoilState } from 'recoil';
import {withAuthenticationRequired} from "@auth0/auth0-react";
import Loading from "../components/Loading";
import {gamesState} from "../api/atoms"



export const GameEditComponent = ({match}) => {

    const [games, ] = useRecoilState(gamesState);

    const gameId = match.params.id;

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
