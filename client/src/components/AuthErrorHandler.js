// components/AuthErrorHandler.js
import React from 'react';
import { Alert } from "reactstrap";
import {useApi} from "../api/useApi";

const AuthErrorHandler = () => {
    const api = useApi();

    const onConsent= (e) => api.handle(e, api.handlerConsent)
    const onLoginAgain= (e) => api.handle(e, api.handlerLoginAgain)

    const { error, audience } = api.state;

    if (error === "consent_required") {
        return (
            <Alert color="warning">
                You need to{" "}
                <a href="#/" className="alert-link" onClick={onConsent}>
                    consent to get access to users api
                </a>
            </Alert>
        );
    } else if (error === "login_required") {
        return (
            <Alert color="warning">
                You need to{" "}
                <a href="#/" className="alert-link" onClick={onLoginAgain}>
                    log in again
                </a>
            </Alert>
        );
    } else if (!audience) {
        return (
            <Alert color="warning">
                {/* The content for the 'audience' error */}
            </Alert>
        );
    }

    return null;
};

export default AuthErrorHandler;
