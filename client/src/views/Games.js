import React, {useEffect} from 'react';
import { useRecoilState } from 'recoil';
import {Button, Table} from "reactstrap";
import {useAuth0, withAuthenticationRequired} from "@auth0/auth0-react";
import Loading from "../components/Loading";
import {useApi} from "../api/useApi";
// import Highlight from "../components/Highlight";
import {gamesState} from "../api/atoms"
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faEdit, faPlay } from '@fortawesome/free-solid-svg-icons';
import {Link} from "react-router-dom";



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
                        <th>#</th>
                        <th>Name</th>
                        <th>Visibility</th>
                        <th>Owner</th>
                        <th>Action</th>
                    </tr>
                    </thead>
                    <tbody>
                    {games.map((game, index) => (
                        <tr key={index}>
                            <td>{game.ID}</td>
                            <td>{game.title}</td>
                            <td>{game.shareState}</td>
                            <td>{game.user.name}</td>
                            <td>
                                <Link to={`/edit/${game.ID}`} className="btn btn-secondary mr-2">
                                    <FontAwesomeIcon icon={faEdit} /> Edit
                                </Link>
                                <Link to={`/play/${game.ID}`} className="btn btn-secondary">
                                    <FontAwesomeIcon icon={faPlay} /> Play
                                </Link>
                            </td>

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