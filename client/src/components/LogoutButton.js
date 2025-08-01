import {useAuth0} from "@auth0/auth0-react";
import {Button} from "reactstrap";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {faSignOutAlt} from "@fortawesome/free-solid-svg-icons";
import React from "react";
import { useMockMode } from "../api/useMockMode";
import { useSetRecoilState } from "recoil";
import { mockAuthState } from "../api/atoms";

export const LogoutButton = () => {
    const {
        logout,
    } = useAuth0();
    
    const mockMode = useMockMode();
    const setIsAuthenticatedMock = useSetRecoilState(mockAuthState);

    const handleLogout = () => {
        if (mockMode) {
            console.log('[MOCK MODE] Logout click intercepted - not calling Auth0');
            setIsAuthenticatedMock(false);
        } else {
            logout({
                returnTo: window.location.origin,
            });
        }
    };

    return (
        <Button color="danger" onClick={handleLogout} className="ml-2">
            <FontAwesomeIcon icon={faSignOutAlt} className="mr-2"/> Logout
        </Button>
    );

}