import { useState } from "react";
import { useAuth0 } from "@auth0/auth0-react";
import { getConfig } from "../config";
import {useRecoilState} from "recoil";
import {errorsState} from "./atoms";

export const useApi = () => {
    const [errors, setErrors ]= useRecoilState(errorsState)

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

    const apiUrl = (endpoint) => {
        endpoint = endpoint.substring(0, 1) === "/" ? endpoint.substring(1) : endpoint;
        return `${config.apiOrigin}/api/${endpoint}`
    }

    const callApi = async (endpoint, data = null, method=null) => {
        try {
            const authorization = !endpoint.startsWith("/public/");
            const token = authorization ? await getAccessTokenSilently() : "";

            const requestOptions = {
                headers: {
                    ...(authorization ? { Authorization: `Bearer ${token}` } : {}),
                    'Content-Type': 'application/json' // Add Content-Type for JSON
                },
                method: method ? method : (data ? 'POST' : 'GET'), // Determine method based on presence of data
            };

            if (data) {
                requestOptions.body = JSON.stringify(data); // Add body if data is present
            }

            console.log("specified method: ", method, "requestOptions: ", requestOptions);
            const response = await fetch(apiUrl(endpoint), requestOptions);

            const responseData = await response.json();

            setState({
                ...state,
                showResult: true,
                apiMessage: responseData,
            });

            if (responseData.type === "error") {
                setErrors([...errors, responseData.error])
            }

            console.log("api response: ", responseData);
            return responseData;
        } catch (error) {
            console.error("api error: ", error);
            setState({
                ...state,
                error: error.error,
            });
        }
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

    return { callApi, apiUrl, handle, handlerConsent, handlerLoginAgain, state, config };
};
