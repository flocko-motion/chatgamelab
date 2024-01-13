import React, {useEffect, useState} from 'react';
import { Container, Row, Col, Input, Button, Badge, Spinner, Toast, ToastHeader, ToastBody } from 'reactstrap';
import {useApi} from "../api/useApi";
import Highlight from "./Highlight";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {faBug} from "@fortawesome/free-solid-svg-icons";


const chapterTypeStory ="story";
const chapterTypeError ="error";
const chapterTypeAction ="player-action";
const chapterTypeLoading ="loading";

const GamePlayer = ({game, sessionHash, debug}) => {
    const api = useApi();

    const [action, setAction] = useState('');
    const [sessionStatus, setSessionStatus] = useState([]);
    const [chapters, setChapters] = useState([]);
    const [actionIdSent, setActionIdSent] = useState(0);
    const [actionIdReceived, setActionIdReceived] = useState(0);

    const receiveChapter = (chapter) => {
        setSessionStatus(chapter.status);
        setChapters(chapters => [...chapters, chapter]);
        if (chapter.actionId) {
            setActionIdReceived(chapter.actionId);
        }
    }

    const submitAction = (action) => {
        setChapters(chapters => [...chapters, {"type": chapterTypeAction, "story": action}]);
        const newActionId = actionIdSent + 1
        setActionIdSent(newActionId);
        api.callApi(`/session/${sessionHash}`, {
            action: "player-action",
            actionId: newActionId,
            message: action,
            status: sessionStatus,
        }).then(chapter => {
            receiveChapter(chapter);
        });
    }

    useEffect(() => {
        if (sessionHash == null || actionIdSent !== 0) return;
        setActionIdSent(1);
        api.callApi(`/session/${sessionHash}`, {
            action: "intro",
            actionId: 1,
        }).then(chapter => {
            receiveChapter(chapter);
        });
    }, []);


    return (
        <Container fluid className="h-100 d-flex flex-column bg-dark">
            <Row className="m-0 p-0">
                <Col>
                    <h1 className="m-0 p-0 text-white">{game.title}</h1>
                    <p  className="m-0 p-0 text-white"><small>Session #{sessionHash}, sent: {actionIdSent}, recv: {actionIdReceived}</small></p>
                </Col>
            </Row>
            <Row className="m-0 p-1">
                <Col>
                    {sessionStatus ? sessionStatus.map((item, index) => {
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
                    {chapters.map((chapter, index) => { return <Chapter key={index} chapter={chapter} debug={Boolean(debug)} /> })}
                    { actionIdSent > actionIdReceived ? <Chapter chapter={{type: chapterTypeLoading }} debug={Boolean(debug)}/> : null }
                </Col>
            </Row>

            {/* Bottom Pane */}
            { actionIdReceived < 1 ? null : <Row className="m-0 p-2">
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
            </Row> }
        </Container>
    );
};

const Chapter = ({chapter, debug}) => {

    const [showDebug, setShowDebug] = useState(false);

    const toggleDebug = () => setShowDebug(!showDebug);

    let debugInfo = ""
    if (chapter.rawOutput) {
        try {
            debugInfo += JSON.stringify(JSON.parse(chapter.rawOutput), null, 4);
        } catch (Exception) {
            debugInfo += chapter.rawOutput;
        }

    }

    return (
        <Toast className="w-100 mt-2">
            <ToastHeader>
                { chapter.type === chapterTypeStory ? "Narrator" : null }
                { chapter.type === chapterTypeAction ? "You" : null }
                { chapter.type === chapterTypeError ? "Error" : null }
                { chapter.type === chapterTypeLoading ? "Loading.." : null }
            </ToastHeader>
            <ToastBody>
                {chapter.type === chapterTypeError ? chapter.error + <br /> + chapter.raw : chapter.story }
                { chapter.type === chapterTypeLoading ? <Spinner color="primary" animation="grow"> </Spinner> : null}

                {debug && debugInfo && (
                    <div className="text-right">
                        <div className="text-right">
                            <FontAwesomeIcon icon={faBug} onClick={toggleDebug} style={{ cursor: 'pointer' }} />
                        </div>
                    </div>
                )}

                {showDebug && (
                    <Highlight>{debugInfo}</Highlight>
                )}
            </ToastBody>

        </Toast>
    );
}

export default GamePlayer;
