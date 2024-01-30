import React, {useEffect} from "react";
import {Router, Route, Switch} from "react-router-dom";
import {useAuth0} from "@auth0/auth0-react";
import initFontAwesome from "./utils/initFontAwesome";
import {useRecoilState} from "recoil";


import Loading from "./components/Loading";
import NavBar from "./components/NavBar";
import Footer from "./components/Footer";
import Home from "./views/Home";
import history from "./utils/history";

import "./App.css";

import {errorsState, gamesState, loadingState, userState} from "./api/atoms";

import {useApi} from "./api/useApi";
import AuthErrorHandler from "./components/AuthErrorHandler";

import Games from "./views/Games";
import GameEdit from "./views/GameEdit";
import GameDebug from "./views/GameDebug";
import GamePlay from "./views/GamePlay";
import Profile from "./views/Profile";
import Errors from "./components/Errors";

initFontAwesome();

const App = () => {

    const {
        user,
        isAuthenticated,
        error
    } = useAuth0();

    const api = useApi();

    const [, setGames] = useRecoilState(gamesState);
    const [, setUserDetails] = useRecoilState(userState);
    const [loading, setLoading] = useRecoilState(loadingState);


    useEffect(() => {
        console.log("user changed: ", user, isAuthenticated);
        if(!isAuthenticated) {
            setGames([]);
            return;
        }
        setLoading(true);
        let loadingCount = 2;
        api.callApi("/user", {...user, openaiKeyPersonal:"-", openaiKeyPublish:"-"})
            .then(userDetails => setUserDetails(userDetails))
            .finally(() => --loadingCount  === 0 && setLoading(false));
        api.callApi("/games")
            .then(games => setGames(games))
            .finally(() => --loadingCount  === 0 && setLoading(false));

    }, [user, isAuthenticated]); // Dependency array ensures the effect runs only when api object changes


    if (error) {
        return <div>Oops... {error.message}</div>;
    }

    if (loading) {
        return <Loading/>;
    }

    return (
            <Router history={history}>
                <div id="app">
                    <NavBar/>
                    <div className="flex-grow-1 overflow-hidden">
                        <AuthErrorHandler/>
                        <Errors />
                        <Switch>
                            <Route path="/" exact component={isAuthenticated ? Games : Home}/>
                            <Route path="/games" component={Games}/>
                            <Route path="/profile" component={Profile}/>
                            <Route path="/edit/:id" component={GameEdit}/>
                            <Route path="/debug/:id" component={GameDebug}/>
                            <Route path="/play/:hash" component={GamePlay}/>
                        </Switch>
                    </div>
                    <Footer/>
                </div>
            </Router>
    );
};

export default App;
