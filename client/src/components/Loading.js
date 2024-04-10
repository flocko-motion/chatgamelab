import React from "react";
import loading from "../assets/loading.svg";
import Errors from "./Errors";

const Loading = () => (
        <div className="spinner">
            <Errors/>
            <img src={loading} alt="Loading"/>
        </div>
);

export default Loading;
