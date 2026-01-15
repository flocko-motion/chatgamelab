/**
 * Navigation utilities and hooks for route management
 */

import { useCallback } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { ROUTES, NAVIGATION_GROUPS, createGameDetailRoute, createGamePlayRoute, createGameEditRoute, createRoomDetailRoute, createRoomJoinRoute } from './routes';
import { navigationLogger } from '../../config/logger';

/**
 * Hook for navigation with type safety
 * Provides convenient methods for navigating to common routes
 */
export const useNavigation = () => {
  const navigate = useNavigate();

  const goToHome = useCallback(() => {
    navigationLogger.debug('Navigating to home', { path: ROUTES.HOME });
    navigate({ to: ROUTES.HOME });
  }, [navigate]);
  const goToDashboard = useCallback(() => {
    navigationLogger.debug('Navigating to dashboard', { path: ROUTES.DASHBOARD });
    navigate({ to: ROUTES.DASHBOARD });
  }, [navigate]);
  const goToProfile = useCallback(() => {
    navigationLogger.debug('Navigating to profile', { path: ROUTES.PROFILE });
    navigate({ to: ROUTES.PROFILE });
  }, [navigate]);
  const goToSettings = useCallback(() => {
    navigationLogger.debug('Navigating to settings', { path: ROUTES.SETTINGS });
    navigate({ to: ROUTES.SETTINGS });
  }, [navigate]);
  const goToLogin = useCallback(() => {
    navigationLogger.debug('Navigating to login', { path: ROUTES.AUTH_LOGIN });
    navigate({ to: ROUTES.AUTH_LOGIN });
  }, [navigate]);
  const goToLogout = useCallback(() => {
    navigationLogger.debug('Navigating to logout', { path: ROUTES.AUTH_LOGOUT as string });
    navigate({ to: ROUTES.AUTH_LOGOUT as string });
  }, [navigate]);
  const goToGames = useCallback(() => {
    navigationLogger.debug('Navigating to games', { path: ROUTES.GAMES as string });
    navigate({ to: ROUTES.GAMES as string });
  }, [navigate]);
  const goToGameCreate = useCallback(() => {
    navigationLogger.debug('Navigating to game create', { path: ROUTES.GAME_CREATE as string });
    navigate({ to: ROUTES.GAME_CREATE as string });
  }, [navigate]);
  const goToRooms = useCallback(() => {
    navigationLogger.debug('Navigating to rooms', { path: ROUTES.ROOMS as string });
    navigate({ to: ROUTES.ROOMS as string });
  }, [navigate]);
  const goToRoomCreate = useCallback(() => {
    navigationLogger.debug('Navigating to room create', { path: ROUTES.ROOM_CREATE as string });
    navigate({ to: ROUTES.ROOM_CREATE as string });
  }, [navigate]);
  const goToDebug = useCallback(() => {
    navigationLogger.debug('Navigating to debug', { path: ROUTES.DEBUG as string });
    navigate({ to: ROUTES.DEBUG as string });
  }, [navigate]);

  const goToGameDetail = useCallback((gameId: string) => {
    const path = createGameDetailRoute(gameId) as string;
    navigationLogger.debug('Navigating to game detail', { gameId, path });
    navigate({ to: path });
  }, [navigate]);
  
  const goToGamePlay = useCallback((gameId: string) => {
    const path = createGamePlayRoute(gameId) as string;
    navigationLogger.debug('Navigating to game play', { gameId, path });
    navigate({ to: path });
  }, [navigate]);
  
  const goToGameEdit = useCallback((gameId: string) => {
    const path = createGameEditRoute(gameId) as string;
    navigationLogger.debug('Navigating to game edit', { gameId, path });
    navigate({ to: path });
  }, [navigate]);
  
  const goToRoomDetail = useCallback((roomId: string) => {
    const path = createRoomDetailRoute(roomId) as string;
    navigationLogger.debug('Navigating to room detail', { roomId, path });
    navigate({ to: path });
  }, [navigate]);
  
  const goToRoomJoin = useCallback((roomId: string) => {
    const path = createRoomJoinRoute(roomId) as string;
    navigationLogger.debug('Navigating to room join', { roomId, path });
    navigate({ to: path });
  }, [navigate]);

  return {
    // Basic navigation
    goToHome,
    goToDashboard,
    goToProfile,
    goToSettings,
    goToLogin,
    goToLogout,
    goToGames,
    goToGameCreate,
    goToRooms,
    goToRoomCreate,
    goToDebug,
    
    // Dynamic navigation
    goToGameDetail,
    goToGamePlay,
    goToGameEdit,
    goToRoomDetail,
    goToRoomJoin,
    
    // Raw navigate for custom routes
    navigate,
  };
};

/**
 * Check if a route belongs to a specific navigation group
 */
export const isRouteInGroup = (pathname: string, group: keyof typeof NAVIGATION_GROUPS): boolean => {
  const result = NAVIGATION_GROUPS[group].some(route => pathname === route || pathname.startsWith(route));
  navigationLogger.debug('Checking route group membership', { pathname, group, result });
  return result;
};

/**
 * Get the current navigation group based on pathname
 */
export const getCurrentNavigationGroup = (pathname: string): keyof typeof NAVIGATION_GROUPS | null => {
  for (const [groupName, routes] of Object.entries(NAVIGATION_GROUPS)) {
    if (routes.some(route => pathname === route || pathname.startsWith(route))) {
      navigationLogger.debug('Found navigation group', { pathname, group: groupName });
      return groupName as keyof typeof NAVIGATION_GROUPS;
    }
  }
  navigationLogger.debug('No navigation group found', { pathname });
  return null;
};
