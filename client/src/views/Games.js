import React, {useEffect} from 'react';
import { useRecoilState } from 'recoil';
import {Button, Table} from "reactstrap";
import {useAuth0, withAuthenticationRequired} from "@auth0/auth0-react";
import Loading from "../components/Loading";
import {useApi} from "../api/useApi";
// import Highlight from "../components/Highlight";
import {gamesState} from "../api/atoms"


export const GamesComponent = () => {
    const api = useApi();

    const [games, ] = useRecoilState(gamesState);


    return (
        <>
            <div className="mb-5">
                <h1>Games</h1>

                <Table striped bordered hover className="mt-4">
                    <thead>
                    <tr>
                        <th>Name</th>
                        <th>Visibility</th>
                        <th>Owner</th>
                        <th>Action</th>
                    </tr>
                    </thead>
                    <tbody>
                    {games.map((game, index) => (
                        <tr key={index}>
                            <td>{game.name}</td>
                            <td>{game.shareState}</td>
                            <td>{game.ownerName}</td>
                            <td><a href={game.editLink}>Edit</a></td>
                        </tr>
                    ))}
                    </tbody>
                </Table>
            </div>
        </>
    );
};

export default withAuthenticationRequired(GamesComponent, {
    onRedirecting: () => <Loading/>,
});
