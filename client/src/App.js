import React, {useEffect, useState} from "react";
import {Router, Route, Switch} from "react-router-dom";
import {useAuth0} from "@auth0/auth0-react";
import initFontAwesome from "./utils/initFontAwesome";
import {useRecoilState} from "recoil";
import { useMockMode } from "./api/useMockMode";


import Loading from "./components/Loading";
import Footer from "./components/Footer";
import Home from "./views/Home";
import history from "./utils/history";

import "./App.css";

import {gamesState, loadingState, userState, mockAuthState} from "./api/atoms";

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
    const mockMode = useMockMode();
    const [isAuthenticatedMock, setIsAuthenticatedMock] = useRecoilState(mockAuthState);

    const {
        user,
        isAuthenticated,
        error
    } = useAuth0();

    const api = useApi();

    const [, setGames] = useRecoilState(gamesState);
    const [, setUserDetails] = useRecoilState(userState);
    const [loading, setLoading] = useRecoilState(loadingState);

    // Handle authentication - either mock or real
    useEffect(() => {
        const actuallyAuthenticated = mockMode ? isAuthenticatedMock : isAuthenticated;
        const actualUser = mockMode ? { sub: 'mock-user', name: 'Mock User' } : user;
        
        console.log("auth changed:", { mockMode, isAuthenticatedMock, isAuthenticated, actuallyAuthenticated });
        
        if (!actuallyAuthenticated) {
            setGames([]);
            return;
        }
        
        setLoading(true);
        let loadingCount = 2;
        api.callApi("/user", {...actualUser, openaiKeyPersonal: "-", openaiKeyPublish: "-"})
            .then(userDetails => setUserDetails(userDetails))
            .finally(() => --loadingCount === 0 && setLoading(false));
        api.callApi("/games")
            .then(games => setGames(games))
            .finally(() => --loadingCount === 0 && setLoading(false));

    }, [mockMode, isAuthenticatedMock, user, isAuthenticated]);


    if (error) {
        return <div>Oops... {error.message}</div>;
    }

    if (loading) {
        return <Loading/>;
    }


    return (
        <Router history={history}>
            <div id="app">
                <div className="flex-grow-1 overflow-hidden">
                    <AuthErrorHandler/>
                    <Errors/>
                    <Switch>
                        {(isAuthenticated || isAuthenticatedMock) && (
                            <>
                                <Route path="/" exact component={Games}/>
                                <Route path="/games" component={Games}/>
                                <Route path="/profile" component={Profile}/>
                                <Route path="/edit/:id" component={GameEdit}/>
                                <Route path="/debug/:id" component={GameDebug}/>
                                <Route path="/play/:hash" component={GamePlay}/>
                            </>
                        )}
                        <Route path="/play/:hash" component={GamePlay}/>
                        <Route path="/" component={Home}/>
                    </Switch>
                </div>
                <Footer/>
            </div>
        </Router>
    );
};

export default App;
