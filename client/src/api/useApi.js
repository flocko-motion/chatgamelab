import { useState } from "react";
import { useAuth0 } from "@auth0/auth0-react";
import { getConfig } from "../config";
import {useRecoilState} from "recoil";
import {errorsState} from "./atoms";
import { useMockMode } from "./useMockMode";
import { getMockResponse, mockImageUrl } from "./mockData";

export const useApi = () => {
    const [errors, setErrors ]= useRecoilState(errorsState)
    const mockMode = useMockMode();

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
        
        // In mock mode, return mock data URLs for image endpoints
        if (mockMode && endpoint.startsWith('image/')) {
            return mockImageUrl;
        }
        
        return `${config.apiOrigin}/api/${endpoint}`
    }

    const callApi = async (endpoint, data = null, method=null) => {
        console.log(`[DEBUG] callApi called: mockMode=${mockMode}, endpoint=${endpoint}`);
        try {
            // If mock mode is enabled, return mock data instead of making real API calls
            if (mockMode) {
                console.log(`[MOCK MODE] Intercepted API call: ${method || (data ? 'POST' : 'GET')} ${endpoint}`);
                const responseData = await getMockResponse(endpoint, data, method || (data ? 'POST' : 'GET'));
                
                setState({
                    ...state,
                    showResult: true,
                    apiMessage: responseData,
                });

                if (responseData.type === "error") {
                    setErrors([...errors, responseData.error])
                }

                console.log("[MOCK MODE] Mock response: ", responseData);
                console.log("[MOCK MODE] Returning early, should NOT reach Auth0 calls");
                return responseData;
            }

            console.log("[DEBUG] NOT in mock mode, proceeding with real API call");

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
        console.log(`[DEBUG] handlerConsent called: mockMode=${mockMode}`);
        if (mockMode) {
            console.log("[MOCK MODE] handlerConsent intercepted - not calling Auth0");
            return;
        }
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
        console.log(`[DEBUG] handlerLoginAgain called: mockMode=${mockMode}`);
        if (mockMode) {
            console.log("[MOCK MODE] handlerLoginAgain intercepted - not calling Auth0");
            return;
        }
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
