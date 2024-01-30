import { ButtonGroup, Col, Row } from "reactstrap";
import { SettingsButton } from "./SettingsButton";
import { LogoutButton } from "./LogoutButton";
import React from "react";
import {LoginButton} from "./LoginButton";
import {useAuth0} from "@auth0/auth0-react";

export const Menu = ({ title, children }) => {
    const {
        isAuthenticated,
    } = useAuth0();

    return (
        <Row className="align-items-center mb-4">
            <Col xs="12" md="6">
                <h1>{title}</h1>
            </Col>
            <Col xs="12" md="6" className="text-md-right">
                <ButtonGroup>
                    {children}
                    {isAuthenticated ? <LogoutButton/> : <LoginButton/>}
                </ButtonGroup>
            </Col>
        </Row>
    );
}
