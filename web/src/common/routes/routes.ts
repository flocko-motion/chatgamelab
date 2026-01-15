/**
 * Centralized route constants for navigation
 * 
 * This file provides type-safe route constants that can be imported
 * throughout the application instead of hardcoding route paths.
 * 
 * Usage:
 *   navigate({ to: ROUTES.DASHBOARD })
 *   <Link to={ROUTES.PROFILE}>Profile</Link>
 */

export const ROUTES = {
  // Core pages
  HOME: '/',
  DASHBOARD: '/dashboard',
  PROFILE: '/profile',
  SETTINGS: '/settings',
  API_KEYS: '/api-keys',

  // Authentication routes
  AUTH_LOGIN: '/auth/login',
  AUTH_LOGOUT: '/auth/logout',
  AUTH_REGISTER: '/auth/register',
  
  // Auth0 specific routes
  AUTH0_CALLBACK: '/auth/login/auth0/callback',
  AUTH0_LOGOUT_CALLBACK: '/auth/logout/auth0/callback',

  // Game routes
  GAMES: '/games',
  GAME_CREATE: '/games/create',
  GAME_DETAIL: '/games/$gameId',
  GAME_PLAY: '/games/$gameId/play',
  GAME_EDIT: '/games/$gameId/edit',

  // Room routes
  ROOMS: '/rooms',
  ROOM_CREATE: '/rooms/create',
  ROOM_DETAIL: '/rooms/$roomId',
  ROOM_JOIN: '/rooms/$roomId/join',

  // Debug routes
  DEBUG: '/debug',
  DEBUG_ERROR: '/debug/error',
} as const;

// Route groups for navigation menus
export const NAVIGATION_GROUPS = {
  MAIN: [ROUTES.HOME, ROUTES.DASHBOARD],
  AUTH: [ROUTES.AUTH_LOGIN, ROUTES.AUTH_REGISTER],
  USER: [ROUTES.PROFILE, ROUTES.SETTINGS, ROUTES.API_KEYS],
  GAMES: [ROUTES.GAMES, ROUTES.GAME_CREATE],
  ROOMS: [ROUTES.ROOMS, ROUTES.ROOM_CREATE],
  DEBUG: [ROUTES.DEBUG],
} as const;

// Type helpers for route params
export type RouteParams = {
  gameId: string;
  roomId: string;
};

// Helper functions for dynamic routes
export const createGameDetailRoute = (gameId: string) => `/games/${gameId}`;
export const createGamePlayRoute = (gameId: string) => `/games/${gameId}/play`;
export const createGameEditRoute = (gameId: string) => `/games/${gameId}/edit`;
export const createRoomDetailRoute = (roomId: string) => `/rooms/${roomId}`;
export const createRoomJoinRoute = (roomId: string) => `/rooms/${roomId}/join`;
