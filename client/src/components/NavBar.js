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
import { useRecoilValue, useSetRecoilState } from "recoil";
import { mockAuthState, cglAuthState } from "../api/atoms";
import { useMockMode } from "../api/useMockMode";

const NavBar = () => {
  const [isOpen, setIsOpen] = useState(false);
  const mockMode = useMockMode();
  const {
    user,
    isAuthenticated,
    loginWithRedirect,
    logout,
  } = useAuth0();
  const toggle = () => setIsOpen(!isOpen);
  const location = useLocation();

  // Use Recoil state for mock and CGL authentication
  const isAuthenticatedMock = useRecoilValue(mockAuthState);
  const setIsAuthenticatedMock = useSetRecoilState(mockAuthState);
  const isAuthenticatedCgl = useRecoilValue(cglAuthState);
  const setIsAuthenticatedCgl = useSetRecoilState(cglAuthState);
  const actuallyAuthenticated = isAuthenticated || isAuthenticatedMock || isAuthenticatedCgl;

  console.log('auth state:', { mockMode, isAuthenticatedMock, isAuthenticatedCgl, isAuthenticated, actuallyAuthenticated, hasCglToken: !!localStorage.getItem('cgl_token') });
  const actualUser = isAuthenticatedMock ?
    { name: 'Mock Developer', picture: 'https://via.placeholder.com/150/0000FF/808080?text=Mock' } :
    (isAuthenticatedCgl ? { name: 'CGL User', picture: 'https://via.placeholder.com/150/0000FF/808080?text=CGL' } : user);

  const handleLogin = () => {
    console.log("handleLogin:mockMode=", mockMode);
    if (mockMode) {
      console.log('[MOCK MODE] NavBar login click intercepted - not calling Auth0');
      setIsAuthenticatedMock(true);
    } else {
      loginWithRedirect({});
    }
  };

  const handleLogout = () => {
    console.log("handleLogout: mockMode=", mockMode, "cglAuth=", isAuthenticatedCgl);
    if (isAuthenticatedCgl) {
      console.log('[CGL] Logout - clearing CGL token');
      localStorage.removeItem('cgl_token');
      setIsAuthenticatedCgl(false);
    } else if (mockMode) {
      console.log('[MOCK MODE] Logout click intercepted - not calling Auth0');
      setIsAuthenticatedMock(false);
    } else {
      logout({
        returnTo: window.location.origin,
      });
    }
  };

  if (location.pathname.startsWith('/play')) {
    return null;
  }

  return (
    <div className="nav-container">
      <Navbar color="light" light container={false}>
        <Container>
          <NavbarToggler onClick={toggle} />
          <Collapse isOpen={isOpen} navbar>
            <Nav className="mr-auto" navbar>
              {!actuallyAuthenticated && (
                <NavItem>
                  <Button
                    id="qsLoginBtn"
                    color="primary"
                    block
                    onClick={handleLogin}
                  >
                    Log in
                  </Button>
                </NavItem>
              )}
              {actuallyAuthenticated && (
                <UncontrolledDropdown nav inNavbar>
                  <DropdownToggle nav caret id="profileDropDown">
                    <img
                      src={actualUser.picture}
                      alt="Profile"
                      className="nav-user-profile rounded-circle"
                      width="50"
                    />
                  </DropdownToggle>
                  <DropdownMenu>
                    <DropdownItem header>{actualUser.name}</DropdownItem>
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
                      onClick={handleLogout}
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
