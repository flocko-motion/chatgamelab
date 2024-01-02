import React, { useState } from "react";
import { Button, Alert } from "reactstrap";
import Highlight from "../components/Highlight";
import { useAuth0, withAuthenticationRequired } from "@auth0/auth0-react";
import { getConfig } from "../config";
import Loading from "../components/Loading";
import {useApi} from "../api/useApi";
import AuthErrorHandler from "../components/AuthErrorHandler";

export const GamesComponent = () => {
  const api = useApi();

  return (
    <>
      <div className="mb-5">

        <h1>Games</h1>
        <p className="lead">
          Ping an external API by clicking the button below.
        </p>

        <p>
          This will call a local API
        </p>


        <Button
          color="primary"
          className="mt-5"
          onClick={api.callApi}
        >
          Ping API
        </Button>
      </div>

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
    </>
  );
};

export default withAuthenticationRequired(GamesComponent, {
  onRedirecting: () => <Loading />,
});
