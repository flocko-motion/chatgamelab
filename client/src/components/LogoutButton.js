import {useAuth0} from "@auth0/auth0-react";
import {Button} from "reactstrap";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {faSignOutAlt} from "@fortawesome/free-solid-svg-icons";
import React from "react";

export const LogoutButton = () => {
    const {
        logout,
    } = useAuth0();

    const logoutWithRedirect = () =>
        logout({
            returnTo: window.location.origin,
        });

    return (
        <Button color="danger" onClick={() => logoutWithRedirect()} className="ml-2">
            <FontAwesomeIcon icon={faSignOutAlt} className="mr-2"/> Logout
        </Button>
    );

}