import React, {useEffect, useState} from 'react';
import { Container, Row, Col, Input, Button, Badge } from 'reactstrap';
import {useApi} from "../api/useApi";

const GamePlayer = ({game, sessionHash}) => {
    const api = useApi();

    const [action, setAction] = useState('');
    const [statusBar, setStatusBar] = useState([]);
    const [chapters, setChapters] = useState([]);

    const receiveChapter = (chapter) => {
        console.log("received chapter: ", chapter);
        setStatusBar(chapter.status);
        setChapters(chapters => [...chapters, chapter]);
    }

    const submitAction = (action) => {
        setChapters(chapters => [...chapters, {"type": "action", "story": action}]);
        api.callApi(`/session/${sessionHash}`, {
            action: "player-action",
            message: action,
        }).then(chapter => {
            receiveChapter(chapter);
        });
    }

    useEffect(() => {
        api.callApi(`/session/${sessionHash}`, {
            action: "intro",
        }).then(chapter => {
            receiveChapter(chapter);
        });
    }, []);


    return (
        <Container fluid className="h-100 d-flex flex-column bg-dark">
            <Row className="m-0 p-1">
                <Col>
                    {statusBar ? statusBar.map((item, index) => {
                        return (
                            <Badge color="light" key={index} className="mr-2">
                                {item.name}: {item.value}
                            </Badge>
                        );
                    }) : null}

                </Col>
            </Row>

            {/* Main Pane */}
            <Row className="flex-grow-1 overflow-auto ml-0 bg-light">
                <Col>
                    <h1>{game.title}</h1>
                    <p><small>Session #{sessionHash}</small></p>
                    {chapters.map((chapter, index) => {
                        return (
                            <div key={index}>
                                <p><b>{chapter.type}</b> {chapter.story}</p>
                            </div>
                        );
                    })}
                </Col>
            </Row>

            {/* Bottom Pane */}
            <Row className="m-0 p-2">
                <Col className="d-flex align-items-center">
                    <Input
                        type="text"
                        placeholder="Enter your action..."
                        className="mr-2"
                        value={action}
                        onChange={(e) => setAction(e.target.value)}
                        onKeyDown={(e) => {
                            if (e.key === 'Enter') {
                                submitAction(action);
                                setAction(''); // Optional: clear input after submit
                            }
                        }}
                    />
                    <Button color="primary" onClick={() => submitAction(action)}>Submit</Button>
                </Col>
            </Row>
        </Container>
    );
};

export default GamePlayer;
