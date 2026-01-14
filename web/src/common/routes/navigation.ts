/**
 * Navigation utilities and hooks for route management
 */

import { useCallback } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { ROUTES, NAVIGATION_GROUPS, createGameDetailRoute, createGamePlayRoute, createGameEditRoute, createRoomDetailRoute, createRoomJoinRoute } from './routes';

/**
 * Hook for navigation with type safety
 * Provides convenient methods for navigating to common routes
 */
export const useNavigation = () => {
  const navigate = useNavigate();

  const goToHome = useCallback(() => navigate({ to: ROUTES.HOME }), [navigate]);
  const goToDashboard = useCallback(() => navigate({ to: ROUTES.DASHBOARD }), [navigate]);
  const goToProfile = useCallback(() => navigate({ to: ROUTES.PROFILE }), [navigate]);
  const goToSettings = useCallback(() => navigate({ to: ROUTES.SETTINGS }), [navigate]);
  const goToLogin = useCallback(() => navigate({ to: ROUTES.AUTH_LOGIN }), [navigate]);
  const goToLogout = useCallback(() => navigate({ to: ROUTES.AUTH_LOGOUT as any }), [navigate]);
  const goToGames = useCallback(() => navigate({ to: ROUTES.GAMES as any }), [navigate]);
  const goToGameCreate = useCallback(() => navigate({ to: ROUTES.GAME_CREATE as any }), [navigate]);
  const goToRooms = useCallback(() => navigate({ to: ROUTES.ROOMS as any }), [navigate]);
  const goToRoomCreate = useCallback(() => navigate({ to: ROUTES.ROOM_CREATE as any }), [navigate]);
  const goToDebug = useCallback(() => navigate({ to: ROUTES.DEBUG as any }), [navigate]);

  const goToGameDetail = useCallback((gameId: string) => 
    navigate({ to: createGameDetailRoute(gameId) as any }), [navigate]);
  
  const goToGamePlay = useCallback((gameId: string) => 
    navigate({ to: createGamePlayRoute(gameId) as any }), [navigate]);
  
  const goToGameEdit = useCallback((gameId: string) => 
    navigate({ to: createGameEditRoute(gameId) as any }), [navigate]);
  
  const goToRoomDetail = useCallback((roomId: string) => 
    navigate({ to: createRoomDetailRoute(roomId) as any }), [navigate]);
  
  const goToRoomJoin = useCallback((roomId: string) => 
    navigate({ to: createRoomJoinRoute(roomId) as any }), [navigate]);

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
  return NAVIGATION_GROUPS[group].some(route => pathname === route || pathname.startsWith(route));
};

/**
 * Get the current navigation group based on pathname
 */
export const getCurrentNavigationGroup = (pathname: string): keyof typeof NAVIGATION_GROUPS | null => {
  for (const [groupName, routes] of Object.entries(NAVIGATION_GROUPS)) {
    if (routes.some(route => pathname === route || pathname.startsWith(route))) {
      return groupName as keyof typeof NAVIGATION_GROUPS;
    }
  }
  return null;
};
