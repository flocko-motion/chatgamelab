import React from "react";
import logo from "../assets/logo.svg";
import {Menu} from "../components/Menu";
import Content from "../components/Content";

const Home = () => {

    return (<Content>
        <Menu title="" />
        <div className="text-center hero my-5">
            <img className="mb-3 app-logo" src={logo} alt="Logo" width="120"/>
            <h1 className="mb-4">
                ChatGameLab
            </h1>

            <p className="lead">
                An educational game, allowing the creation of GPT-4 based text-adventure games.
            </p>
        </div>
    </Content>);
}

export default Home;
