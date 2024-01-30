import {useAuth0} from "@auth0/auth0-react";
import {Button} from "reactstrap";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {faSignInAlt} from "@fortawesome/free-solid-svg-icons";
import React from "react";

export const LoginButton = () => {
    const {
        loginWithRedirect,
    } = useAuth0();

    return (
        <Button color="primary" onClick={() => loginWithRedirect({})} className="ml-2">
            <FontAwesomeIcon icon={faSignInAlt} className="mr-2"/> Login
        </Button>
    );

}