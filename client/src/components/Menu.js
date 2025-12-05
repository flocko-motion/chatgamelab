import { ButtonGroup, Col, Row } from "reactstrap";
import { SettingsButton } from "./SettingsButton";
import { LogoutButton } from "./LogoutButton";
import React from "react";
import {LoginButton} from "./LoginButton";
import {useAuth0} from "@auth0/auth0-react";
import { useMockMode } from "../api/useMockMode";
import { useRecoilValue } from "recoil";
import { mockAuthState } from "../api/atoms";

export const Menu = ({ title, children }) => {
    const {
        isAuthenticated,
    } = useAuth0();
    
    const mockMode = useMockMode();
    const isAuthenticatedMock = useRecoilValue(mockAuthState);
    const actuallyAuthenticated = isAuthenticated || isAuthenticatedMock;

    return (
        <Row className="align-items-center mb-4">
            <Col xs="12" md="6">
                <h1>{title}</h1>
            </Col>
            <Col xs="12" md="6" className="text-md-right">
                <ButtonGroup>
                    {children}
                    {actuallyAuthenticated ? <LogoutButton/> : <LoginButton/>}
                </ButtonGroup>
            </Col>
        </Row>
    );
}
