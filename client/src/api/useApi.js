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

    const callApi = async () => {
        try {
            const token = await getAccessTokenSilently();

            const response = await fetch(`${config.apiOrigin}/api/external`, {
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
        }    };

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

        await callApi();
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

        await callApi();    };

    const handle = (e, fn) => {
        e.preventDefault();
        fn();
    };

    return { callApi, handle, handlerConsent, handlerLoginAgain, state, config };
};
