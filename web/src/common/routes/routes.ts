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

  // Invite routes (public - no auth required)
  INVITES: '/invites',

  // Participant workshop route
  MY_WORKSHOP: '/my-workshop',

  // My Games (user's own games - create/edit/play)
  MY_GAMES: '/my-games',
  MY_GAME_DETAIL: '/my-games/$gameId',
  MY_GAME_CREATE: '/my-games/create',

  // All Games (browse all public + own games)
  ALL_GAMES: '/games',
  
  // Game play routes (actual gameplay)
  GAME_PLAY: '/games/$gameId/play',

  // Sessions (kept for direct session access)
  SESSIONS: '/sessions',
  SESSION_DETAIL: '/sessions/$sessionId',

  // Room routes
  ROOMS: '/rooms',
  ROOM_CREATE: '/rooms/create',
  ROOM_DETAIL: '/rooms/$roomId',
  ROOM_JOIN: '/rooms/$roomId/join',

  // Debug routes
  DEBUG: '/debug',
  DEBUG_ERROR: '/debug/error',

  // Admin routes
  ADMIN_ORGANIZATIONS: '/admin/organizations',
  ADMIN_USERS: '/admin/users',

  // Organization routes
  MY_ORGANIZATION: '/my-organization',
  MY_ORGANIZATION_API_KEYS: '/my-organization/api-keys',
  MY_ORGANIZATION_WORKSHOPS: '/my-organization/workshops',
} as const;

// Route groups for navigation menus
export const NAVIGATION_GROUPS = {
  MAIN: [ROUTES.HOME, ROUTES.DASHBOARD],
  AUTH: [ROUTES.AUTH_LOGIN, ROUTES.AUTH_REGISTER],
  USER: [ROUTES.PROFILE, ROUTES.SETTINGS, ROUTES.API_KEYS],
  MY_GAMES: [ROUTES.MY_GAMES],
  ALL_GAMES: [ROUTES.ALL_GAMES],
  ROOMS: [ROUTES.ROOMS, ROUTES.ROOM_CREATE],
  DEBUG: [ROUTES.DEBUG],
  ADMIN: [ROUTES.ADMIN_ORGANIZATIONS, ROUTES.ADMIN_USERS],
} as const;

// Type helpers for route params
export type RouteParams = {
  gameId: string;
  roomId: string;
};

// Helper functions for dynamic routes
export const createGameDetailRoute = (gameId: string) => `/my-games/${gameId}`;
export const createGamePlayRoute = (gameId: string) => `/games/${gameId}/play`;
export const createGameEditRoute = (gameId: string) => `/my-games/${gameId}/edit`;
export const createRoomDetailRoute = (roomId: string) => `/rooms/${roomId}`;
export const createRoomJoinRoute = (roomId: string) => `/rooms/${roomId}/join`;
