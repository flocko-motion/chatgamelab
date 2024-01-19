import React, {useState} from 'react';
import {useRecoilState} from 'recoil';
import {Button, Table, Modal, ModalHeader, ModalBody, ModalFooter} from "reactstrap";
import {withAuthenticationRequired} from "@auth0/auth0-react";
import Loading from "../components/Loading";
import {useApi} from "../api/useApi";
import {gamesState} from "../api/atoms"
import {FontAwesomeIcon} from '@fortawesome/react-fontawesome';
import {faEdit, faPlay, faBug, faTrash} from '@fortawesome/free-solid-svg-icons';
import {Link} from "react-router-dom";
import {SharePlayUri, SharePlayUrl, DebugUri, EditUri} from "../utils/urls";
import Content from "../components/Content";


export const GamesComponent = () => {
    const api = useApi();

    const [games, setGames] = useRecoilState(gamesState);

    const handleCreateGame = () => {
        const timestamp = new Date().getTime().toString();
        const last8Digits = timestamp.slice(-8);
        const title = "New " + last8Digits;

        api.callApi(`/game/new`, {
            title: title,
        }).then(game => {
            window.location.href = EditUri(game.id);
        });
    }

    const [isModalOpen, setIsModalOpen] = useState(false);
    const [selectedGameId, setSelectedGameId] = useState(null);

    const handleDeleteGame = (gameId) => {
        setSelectedGameId(gameId);
        setIsModalOpen(true);
    };

    const confirmDeletion = () => {

        api.callApi(`/game/${selectedGameId}`, null, 'DELETE')
            .then(() => {
                // Update games state
                setIsModalOpen(false);
                setSelectedGameId(null);
                // Additional logic after deletion
                api.callApi("/games").then(games => setGames(games));
            });
    };


    return (
        <>
            <Content>
                <h1>Games</h1>

                <Table striped bordered hover className="mt-4">
                    <thead>
                    <tr>
                        <th>#</th>
                        <th>Name</th>
                        <th>Owner</th>
                        <th>Action</th>
                        <th>Public URL</th>
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
                                    <FontAwesomeIcon icon={faEdit}/>
                                </Link>
                                <Link to={DebugUri(game.id)} className="btn btn-secondary mr-2">
                                    <FontAwesomeIcon icon={faBug}/>
                                </Link>
                                <a href={SharePlayUri(game.sharePlayHash)} target="_blank"
                                   className="btn btn-secondary mr-2" rel="noreferrer">
                                    <FontAwesomeIcon icon={faPlay}/>
                                </a>
                                <Button color="danger" onClick={() => handleDeleteGame(game.id)}>
                                    <FontAwesomeIcon icon={faTrash}/>
                                </Button>
                            </td>
                            <td>
                                {game.sharePlayActive ?
                                    <a href={SharePlayUrl(game.sharePlayHash)}
                                       target="_blank">{SharePlayUrl(game.sharePlayHash)}</a>
                                    : "not published"}
                            </td>
                        </tr>
                    ))}
                    </tbody>
                </Table>
                <Button color="primary" onClick={handleCreateGame}>Create New Game</Button>

            </Content>

            <Modal isOpen={isModalOpen} toggle={() => setIsModalOpen(false)}>
                <ModalHeader toggle={() => setIsModalOpen(false)}>
                    Confirm Deletion
                </ModalHeader>
                <ModalBody>
                    Do you really want to delete this game?
                </ModalBody>
                <ModalFooter>
                    <Button color="danger" onClick={confirmDeletion}>Delete</Button>{' '}
                    <Button color="secondary" onClick={() => setIsModalOpen(false)}>Cancel</Button>
                </ModalFooter>
            </Modal>
        </>
    );
};

export default withAuthenticationRequired(GamesComponent, {
    onRedirecting: () => <Loading/>,
});
