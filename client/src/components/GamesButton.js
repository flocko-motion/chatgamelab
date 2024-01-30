import {useAuth0} from "@auth0/auth0-react";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {faList} from "@fortawesome/free-solid-svg-icons";
import {Button} from "reactstrap";
import React from "react";
import {useHistory} from "react-router-dom";

export const GamesButton = () => {
    const {
        isAuthenticated,
    } = useAuth0();

    const history = useHistory();

    // Note: to="/profile"
    return isAuthenticated ? (
        <Button color="secondary" onClick={() => history.push("/games")} className="ml-2">
            <FontAwesomeIcon icon={faList} className="mr-2"/> Games
        </Button>
    ) : null;

}