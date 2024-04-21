import React from "react";
import {version} from "../version.js";

const Footer = () => (
    <footer className="bg-light p-1 text-center">
            Login by <a href="https://auth0.com">Auth0</a> |
            Programmed by <a href="https://omnitopos.net">omnitopos.net</a> |
        Produced by <a href="https://tausend-medien.de">tausend-medien.de</a> | v{version}
    </footer>
);

export default Footer;
