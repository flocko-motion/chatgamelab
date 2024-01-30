import React, { useState } from "react";
import { NavLink as RouterNavLink } from "react-router-dom";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { useLocation } from 'react-router-dom';

import {
  Collapse,
  Container,
  Navbar,
  NavbarToggler,
  Nav,
  NavItem,
  Button,
  UncontrolledDropdown,
  DropdownToggle,
  DropdownMenu,
  DropdownItem,
} from "reactstrap";

import { useAuth0 } from "@auth0/auth0-react";

const NavBar = () => {
  const [isOpen, setIsOpen] = useState(false);
  const {
    user,
    isAuthenticated,
    loginWithRedirect,
    logout,
  } = useAuth0();
  const toggle = () => setIsOpen(!isOpen);
  const location = useLocation();

  const logoutWithRedirect = () =>
      logout({
        returnTo: window.location.origin,
      });

  if(location.pathname.startsWith('/play')) {
    return null;
  }

  return (
      <div className="nav-container">
        <Navbar color="light" light container={false}>
          <Container>
            <NavbarToggler onClick={toggle} />
            <Collapse isOpen={isOpen} navbar>
              <Nav className="mr-auto" navbar>
                {!isAuthenticated && (
                    <NavItem>
                      <Button
                          id="qsLoginBtn"
                          color="primary"
                          block
                          onClick={() => loginWithRedirect({})}
                      >
                        Log in
                      </Button>
                    </NavItem>
                )}
                {isAuthenticated && (
                    <UncontrolledDropdown nav inNavbar>
                      <DropdownToggle nav caret id="profileDropDown">
                        <img
                            src={user.picture}
                            alt="Profile"
                            className="nav-user-profile rounded-circle"
                            width="50"
                        />
                      </DropdownToggle>
                      <DropdownMenu>
                        <DropdownItem header>{user.name}</DropdownItem>
                        <DropdownItem
                            tag={RouterNavLink}
                            to="/profile"
                            className="dropdown-profile"
                            activeClassName="router-link-exact-active"
                        >
                          <FontAwesomeIcon icon="user" className="mr-3" /> Profile
                        </DropdownItem>
                        <DropdownItem
                            id="qsLogoutBtn"
                            onClick={() => logoutWithRedirect()}
                        >
                          <FontAwesomeIcon icon="power-off" className="mr-3" /> Log out
                        </DropdownItem>
                      </DropdownMenu>
                    </UncontrolledDropdown>
                )}
              </Nav>
            </Collapse>
          </Container>
        </Navbar>
      </div>
  );
};

export default NavBar;
