import React from "react";
import {Menu} from "../components/Menu";
import Content from "../components/Content";

const Home = () => {

    return (<Content>
        <Menu title="" />
        <div className="text-center hero my-5">
            <img className="mb-3 app-logo" src={`${process.env.PUBLIC_URL}/logo.png`} alt="Logo" width="60%"/>
            <p className="lead">
                An educational game, allowing the creation of GPT-4 based text-adventure games.
            </p>
        </div>
    </Content>);
}

export default Home;
