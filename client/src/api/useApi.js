import { useState } from "react";
import { useAuth0 } from "@auth0/auth0-react";
import { getConfig } from "../config";

export const useApi = () => {
    const [state, setState] = useState({
        showResult: false,
        apiMessage: "",
        error: null,
    });

    const {
        getAccessTokenSilently,
        loginWithPopup,
        getAccessTokenWithPopup,
    } = useAuth0();

    const config = getConfig();

    const callApi = async (endpoint) => {
        try {
            const token = await getAccessTokenSilently();

            endpoint = endpoint.substring(0,1) === "/" ? endpoint.substring(1) : endpoint;

            const response = await fetch(`${config.apiOrigin}/api/${endpoint}`, {
                headers: {
                    Authorization: `Bearer ${token}`,
                },
            });

            const responseData = await response.json();

            setState({
                ...state,
                showResult: true,
                apiMessage: responseData,
            });
        } catch (error) {
            setState({
                ...state,
                error: error.error,
            });
        }

        const gamesStateExample = [
            {name: "Game Foo", shareState: "public", editLink: "/edit/game1", ownerName: "Alice"},
            {name: "Game Bar", shareState: "private", editLink: "/edit/game2", ownerName: "Bob"},
            // ... more games
        ];

        return gamesStateExample
    };

    const handlerConsent = async () => {
        try {
            await getAccessTokenWithPopup();
            setState({
                ...state,
                error: null,
            });
        } catch (error) {
            setState({
                ...state,
                error: error.error,
            });
        }
        console.log("handlerConsent - what now?")
        await callApi("/external");
    };

    const handlerLoginAgain = async () => {
        try {
            await loginWithPopup();
            setState({
                ...state,
                error: null,
            });
        } catch (error) {
            setState({
                ...state,
                error: error.error,
            });
        }
        console.log("handlerLoginAgain - why now?")
        await callApi("/external");    };

    const handle = (e, fn) => {
        e.preventDefault();
        fn();
    };

    return { callApi, handle, handlerConsent, handlerLoginAgain, state, config };
};
