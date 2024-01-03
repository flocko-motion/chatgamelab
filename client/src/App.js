import React, {useEffect} from "react";
import {Router, Route, Switch} from "react-router-dom";
import {Container} from "reactstrap";
import {useAuth0} from "@auth0/auth0-react";
import initFontAwesome from "./utils/initFontAwesome";
import {useRecoilState} from "recoil";

import Loading from "./components/Loading";
import NavBar from "./components/NavBar";
import Footer from "./components/Footer";
import Home from "./views/Home";
import history from "./utils/history";

import "./App.css";

import {gamesState} from "./api/atoms";

import {useApi} from "./api/useApi";
import AuthErrorHandler from "./components/AuthErrorHandler";

import Games from "./views/Games";
import Profile from "./views/Profile";

initFontAwesome();

const App = () => {

    const {
        user,
        isAuthenticated,
        isLoading,
        error
    } = useAuth0();

    const api = useApi();

    const [, setGames] = useRecoilState(gamesState);

    useEffect(() => {
        console.log("user changed: ", user, isAuthenticated);
        if(!isAuthenticated) {
            setGames([]);
            return;
        }
        api.callApi("/games").then(games => setGames(games));
    }, [user, isAuthenticated]); // Dependency array ensures the effect runs only when api object changes


    if (error) {
        return <div>Oops... {error.message}</div>;
    }

    if (isLoading) {
        return <Loading/>;
    }


    return (
            <Router history={history}>
                <div id="app" className="d-flex flex-column h-100">
                    <NavBar/>
                    <Container className="flex-grow-1 mt-5">
                        <AuthErrorHandler/>

                        <Switch>
                            <Route path="/" exact component={Home}/>
                            <Route path="/profile" component={Profile}/>
                            <Route path="/games" component={Games}/>
                        </Switch>
                    </Container>
                    <Footer/>
                </div>
            </Router>
    );
};

export default App;
