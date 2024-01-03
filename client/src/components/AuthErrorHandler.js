// components/AuthErrorHandler.js
import React from 'react';
import { Alert } from "reactstrap";
import {useApi} from "../api/useApi";
import Highlight from "./Highlight";

const AuthErrorHandler = () => {
    const api = useApi();

    const onConsent = (e) => api.handle(e, api.handlerConsent)
    const onLoginAgain = (e) => api.handle(e, api.handlerLoginAgain)


    if (api.state.error === "consent_required") {
        return (
            <Alert color="warning">
                You need to{" "}
                <a href="#/" className="alert-link" onClick={onConsent}>
                    consent to get access to users api
                </a>
            </Alert>
        );
    } else if (api.state.error === "login_required") {
        return (
            <Alert color="warning">
                You need to{" "}
                <a href="#/" className="alert-link" onClick={onLoginAgain}>
                    log in again
                </a>
            </Alert>
        );
    } else if (!api.config.audience) {
        return (
            <Alert color="warning">
                No 'audience' claim found in your configuration.
            </Alert>
        );
    }

    if(!api.state.showResult) return null;

    return (
        <div className="result-block-container">
            {api.state.showResult && (
                <div className="result-block" data-testid="api-result">
                    <h6 className="muted">Result</h6>
                    <Highlight>
                        <span>{JSON.stringify(api.state.apiMessage, null, 2)}</span>
                    </Highlight>
                </div>
            )}
        </div>
    );
};

export default AuthErrorHandler;
