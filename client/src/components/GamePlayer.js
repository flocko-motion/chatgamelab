import React, {useEffect, useState} from 'react';
import { Row, Col, Input, Button, Badge, Spinner, Toast, ToastHeader, ToastBody } from 'reactstrap';
import {useApi} from "../api/useApi";
import Highlight from "./Highlight";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {faEye} from "@fortawesome/free-solid-svg-icons";
import loading from "../assets/loading.svg";
import {Menu} from "./Menu";
import {GamesButton} from "./GamesButton";
import Content from "./Content";

const chapterTypeStory ="story";
const chapterTypeError ="error";
const chapterTypeAction ="player-action";
const chapterTypeLoading ="loading";

const GamePlayer = ({game, sessionHash, debug, publicSession}) => {
    const api = useApi();

    const [action, setAction] = useState('');
    const [sessionStatus, setSessionStatus] = useState([]);
    const [chapters, setChapters] = useState([]);
    const [chapterIdSent, setChapterIdSent] = useState(0);
    const [chapterIdReceived, setChapterIdReceived] = useState(0);

    const receiveChapter = (chapter) => {
        console.log("got chapter", chapter);
        setSessionStatus(chapter.status);
        setChapters(chapters => [...chapters, chapter]);
        if (chapter.chapterId) {
            setChapterIdReceived(chapter.chapterId);
        }
    }

    const submitAction = (action) => {
        setChapters(chapters => [...chapters, {"type": chapterTypeAction, "story": action}]);
        const newChapterId = chapterIdSent + 1
        setChapterIdSent(newChapterId);
        api.callApi((publicSession ? '/public' : '') + `/session/${sessionHash}`, {
            action: "player-action",
            chapterId: newChapterId,
            message: action,
            status: sessionStatus,
        }).then(chapter => {
            receiveChapter(chapter);
        });
    }

    useEffect(() => {
        if (sessionHash == null || chapterIdSent !== 0) return;
        setChapterIdSent(1);
        api.callApi((publicSession ? '/public' : '') +`/session/${sessionHash}`, {
            action: "intro",
            chapterId: 1,
        }).then(chapter => {
            receiveChapter(chapter);
        });
    }, []);


    return (
        <Content fluid className="h-100 d-flex flex-column">
            <Menu title={game.title}>
                <GamesButton />
            </Menu>
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
                <Col className="pb-4">
                    {chapters.map((chapter, index) => { return <Chapter key={index} chapter={chapter} debug={Boolean(debug)} /> })}
                    { chapterIdSent > chapterIdReceived ? <Chapter chapter={{type: chapterTypeLoading }} debug={Boolean(debug)}/> : null }
                </Col>
            </Row>

            {/* Bottom Pane */}
            { chapterIdReceived < 1 ? null : <Row className="m-0 p-2">
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
        </Content>
    );
};

const Chapter = ({chapter, debug}) => {

    const api = useApi();

    const [showDebug, setShowDebug] = useState(false);

    const toggleDebug = () => setShowDebug(!showDebug);

    return (
        <Toast className="w-100 mt-2">
            <ToastHeader>
                { chapter.type === chapterTypeStory ? "Narrator" : null }
                { chapter.type === chapterTypeAction ? "You" : null }
                { chapter.type === chapterTypeError ? "Error" : null }
                { chapter.type === chapterTypeLoading ? "Writing story, please be patient.." : null }
            </ToastHeader>
            <ToastBody>
                {chapter.type === chapterTypeError && chapter.error + <br /> + chapter.raw }
                <ChapterContent chapter={chapter} />

                {debug && (chapter.rawInput || chapter.rawOutput)  && (
                    <div className="text-right">
                        <div className="text-right">
                            <FontAwesomeIcon icon={faEye} onClick={toggleDebug} style={{ cursor: 'pointer' }} />
                        </div>
                    </div>
                )}

                {showDebug && chapter.assistantInstructions && <><p>GPT Instructions:</p><Highlight>{ chapter.assistantInstructions }</Highlight></> }
                {showDebug && chapter.rawInput && <><p>GPT Input:</p><Highlight>{ beautifyJson(chapter.rawInput) }</Highlight></> }
                {showDebug && chapter.rawOutput && <><p>GPT Output:</p><Highlight>{ beautifyJson(chapter.rawOutput) }</Highlight></>  }
                {showDebug && chapter.image && <><p>GPT Generated Image Prompt:</p><Highlight>{ chapter.image }</Highlight></> }

            </ToastBody>
        </Toast>
    );
}

const ChapterContent = ({chapter}) => {
    const api = useApi();

    if (chapter.type === chapterTypeLoading) {
        return <Spinner color="primary" animation="grow"> </Spinner>
    }

    if (!chapter.story) {
        return null
    }

    if (chapter.type === chapterTypeAction) {
        return (
            <div className="text-left">
                {chapter.story}
            </div>
        );
    }

    return (
        <>
            <div className="text-left">
                <img
                    src={api.apiUrl(`/image/${chapter.sessionHash}/${chapter.chapterId}`)}
                    alt=""
                    className="float-left mr-2"
                    style={{
                        width: '256px',
                        height: '256px',
                        backgroundColor: '#eaeaea', // placeholder color
                        // or use a background image URL
                        background: `url(${loading}) no-repeat center center`,
                        backgroundSize: 'cover',                    }}
                />
                {chapter.story}
            </div>
            <div className="clearfix"></div>
            {/* Content that follows won't float around the image */}
        </>
    );
}

const beautifyJson = (json) => {
    try {
        return JSON.stringify(JSON.parse(json), null, 2);
    } catch (Exception) {
        return json;
    }
}

export default GamePlayer;
