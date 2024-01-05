import React, { useState } from "react";
import { Container, Row, Col, Form, FormGroup, Label, Input, Button } from "reactstrap";
import { useAuth0, withAuthenticationRequired } from "@auth0/auth0-react";
import Highlight from "../components/Highlight";
import Loading from "../components/Loading";
import {useRecoilState} from "recoil";
import {userState} from "../api/atoms";

export const ProfileComponent = () => {
    const { user } = useAuth0();
    const [userDetails, setUserDetails] = useRecoilState(userState);
    const [personalKey, setPersonalKey] = useState(user.openaiKeyPersonal || ''); // Assuming these keys are part of the user object
    const [publishKey, setPublishKey] = useState(user.openaiKeyPublish || '');

    const handleSave = () => {
        // Implement the logic to save the keys
        console.log('Saving keys:', personalKey, publishKey);
        // This could involve making an API call to your backend to store the keys
    };

    return (
        <Container className="mb-5">
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
                            <Label for="personalKey">Personal Key</Label>
                            <small className="form-text text-muted">
                                OpenAI API key to be used for yourself playing your own games.
                            </small>
                            <Input
                                type="text"
                                id="personalKey"
                                value={personalKey}
                                onChange={(e) => setPersonalKey(e.target.value)}
                            />
                        </FormGroup>
                        <FormGroup>
                            <Label for="publishKey">Publishing Key</Label>
                            <small className="form-text text-muted">
                                OpenAI API key to be used for others playing your published games.
                            </small>
                            <Input
                                type="text"
                                id="publishKey"
                                value={publishKey}
                                onChange={(e) => setPublishKey(e.target.value)}
                            />
                        </FormGroup>
                        <Button color="primary" onClick={handleSave}>Save</Button>
                    </Form>
                </Col>
            </Row>
            <Highlight>{JSON.stringify(user, null, 2)}</Highlight>
            <Highlight>{JSON.stringify(userDetails, null, 2)}</Highlight>
        </Container>
    );
};

export default withAuthenticationRequired(ProfileComponent, {
    onRedirecting: () => <Loading />,
});
