import React from "react";

import Splash from "../components/Splash";
import Content from "../components/Content";
import {Button, NavItem} from "reactstrap";
import {useAuth0} from "@auth0/auth0-react";
import logo from "../assets/logo.svg";

const Home = () => {
    const {
        isAuthenticated,
        loginWithRedirect,
    } = useAuth0();

    return (<div className="text-center hero my-5">
            <img className="mb-3 app-logo" src={logo} alt="Logo" width="120" />
            <h1 className="mb-4">
                ChatGameLab
            </h1>


            <p className="lead">
                An educational game, allowing the creation of GPT-4 based text-adventure games.
            </p>
        {!isAuthenticated && (
            <Button
                id="qsLoginBtn"
                color="primary"
                onClick={() => loginWithRedirect({})}
            >
                Log in
            </Button>
        )}
    </div>)
}

export default Home;
