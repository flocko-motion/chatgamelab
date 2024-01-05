import React from "react";
import { Container, Row, Col, Form, FormGroup, Label, Input, Button } from "reactstrap";
import Content from "../components/Content";
import { useAuth0, withAuthenticationRequired } from "@auth0/auth0-react";
import Loading from "../components/Loading";
import {useRecoilState} from "recoil";
import {userState} from "../api/atoms";

import {useApi} from "../api/useApi";

export const ProfileComponent = () => {
    const api = useApi();
    const { user } = useAuth0();
    const [userDetails, setUserDetails] = useRecoilState(userState);

    const setPersonalKey = (key) => {
        setUserDetails({...userDetails, openaiKeyPersonal: key});
    }

    const setPublishKey = (key) => {
        setUserDetails({...userDetails, openaiKeyPublish: key});
    }

    const handleSave = () => {
        const data = userDetails;
        setUserDetails(null);
        api.callApi("/user", data)
            .then(data => setUserDetails(data));
    };

    if (!user || !userDetails) {
        return <Loading />;
    }

    return (
        <Content>
            <Row className="align-items-center profile-header mb-5 text-center text-md-left">
                <Col md={2}>
                    <img
                        src={user.picture}
                        alt="Profile"
                        className="rounded-circle img-fluid profile-picture mb-3 mb-md-0"
                    />
                </Col>
                <Col md>
                    <h2>{user.name}</h2>
                    <p className="lead text-muted">{user.email}</p>
                </Col>
            </Row>
            <Row>
                <Col lg={12}>
                    <Form>
                        <FormGroup>
                            <Label for="personalKey">Private Playing Key</Label>
                            <small className="form-text text-muted">
                                OpenAI API key to be used for yourself playing your own games while logged in.
                            </small>
                            <Input
                                type="text"
                                id="personalKey"
                                value={userDetails.openaiKeyPersonal}
                                onChange={(e) => setPersonalKey(e.target.value)}
                            />
                        </FormGroup>
                        <FormGroup>
                            <Label for="publishKey">Public Playing Key</Label>
                            <small className="form-text text-muted">
                                OpenAI API key to be used for others playing your games via the public URL.
                            </small>
                            <Input
                                type="text"
                                id="publishKey"
                                value={userDetails.openaiKeyPublish}
                                onChange={(e) => setPublishKey(e.target.value)}
                            />
                        </FormGroup>
                        <Button color="primary" onClick={handleSave}>Save</Button>
                    </Form>
                </Col>
            </Row>
        </Content>
    );
};

export default withAuthenticationRequired(ProfileComponent, {
    onRedirecting: () => <Loading />,
});
