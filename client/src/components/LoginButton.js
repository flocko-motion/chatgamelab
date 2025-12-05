import {useAuth0} from "@auth0/auth0-react";
import {Button} from "reactstrap";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {faSignInAlt} from "@fortawesome/free-solid-svg-icons";
import React from "react";
import { useMockMode } from "../api/useMockMode";
import { useSetRecoilState } from "recoil";
import { mockAuthState } from "../api/atoms";

export const LoginButton = () => {
    const {
        loginWithRedirect,
    } = useAuth0();
    
    const mockMode = useMockMode();
    const setIsAuthenticatedMock = useSetRecoilState(mockAuthState);

    const handleLogin = () => {
        if (mockMode) {
            console.log('[MOCK MODE] Login click intercepted - not calling Auth0');
            setIsAuthenticatedMock(true);
        } else {
            loginWithRedirect({});
        }
    };

    return (
        <Button color="primary" onClick={handleLogin} className="ml-2">
            <FontAwesomeIcon icon={faSignInAlt} className="mr-2"/> Login
        </Button>
    );

}