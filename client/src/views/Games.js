import React, {useEffect} from 'react';
import { useRecoilState } from 'recoil';
import {Button, Table} from "reactstrap";
import {useAuth0, withAuthenticationRequired} from "@auth0/auth0-react";
import Loading from "../components/Loading";
import {useApi} from "../api/useApi";
// import Highlight from "../components/Highlight";
import {gamesState} from "../api/atoms"
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faEdit, faPlay, faBug } from '@fortawesome/free-solid-svg-icons';
import {Link} from "react-router-dom";
import {SharePlayUri, SharePlayUrl, DebugUri, EditUri} from "../utils/urls";



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
                        <th>Owner</th>
                        <th>Action</th>
                        <th>Play URL</th>
                    </tr>
                    </thead>
                    <tbody>
                    {games.map((game, index) => (
                        <tr key={index}>
                            <td>{game.id}</td>
                            <td>{game.title}</td>
                            <td>{game.userName}</td>
                            <td>
                                <Link to={EditUri(game.id)} className="btn btn-secondary mr-2">
                                    <FontAwesomeIcon icon={faEdit} /> Edit
                                </Link>
                                <Link to={DebugUri(game.id)} className="btn btn-secondary mr-2">
                                    <FontAwesomeIcon icon={faBug} /> Debug
                                </Link>
                                <a href={SharePlayUri(game.sharePlayHash)} target="_blank" className="btn btn-secondary">
                                    <FontAwesomeIcon icon={faPlay} /> Play
                                </a>
                            </td>
                            <td>
                                {game.sharePlayActive ?
                                    <a href={SharePlayUrl(game.sharePlayHash)} target="_blank">{SharePlayUrl(game.sharePlayHash)}</a>
                                    : "not published"}
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
