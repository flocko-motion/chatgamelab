import React from "react";
import {
    Container,
    Card,
    CardBody,
    Row,
    Col,
    Form,
    FormGroup,
    Label,
    Input,
    Button,
    ButtonGroup,
    CardHeader
} from "reactstrap";
import Content from "../components/Content";
import { useAuth0 } from "@auth0/auth0-react";
import Loading from "../components/Loading";
import {withMockAwareAuth} from "../utils/withMockAwareAuth";
import {useRecoilState} from "recoil";
import {userState} from "../api/atoms";

import {useApi} from "../api/useApi";
import {LogoutButton} from "../components/LogoutButton";
import {GamesButton} from "../components/GamesButton";
import {Menu} from "../components/Menu";

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
            <Menu title="Settings">
                <GamesButton/>
            </Menu>
            <Card className="mb-4">
                <CardHeader>
                    <h3>{user.name}</h3>
                    <p className="lead text-muted">{user.email}</p>
                </CardHeader>
                <CardBody>
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
                </CardBody>
            </Card>
            <Row>
                <Col lg={12}>

                </Col>
            </Row>
        </Content>
    );
};

export default withMockAwareAuth(ProfileComponent, {
    onRedirecting: () => <Loading />,
});
