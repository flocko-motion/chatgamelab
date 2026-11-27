/**
 * Centralized query keys for TanStack Query.
 *
 * Use these keys consistently across all hooks and components to:
 * - Avoid key collisions
 * - Enable proper cache invalidation
 * - Improve maintainability and type safety
 *
 * Pattern:
 * - Static keys: `keyName: ['key-name'] as const`
 * - Parameterized keys: `keyName: (param) => ['key-name', param] as const`
 */

export const queryKeys = {
  // API Keys
  apiKeys: ["apiKeys"] as const,
  apiKeyShares: ["apiKeyShares"] as const,
  institutionApiKeys: (institutionId: string) =>
    ["institutionApiKeys", institutionId] as const,
  apiKeyGameShares: (shareId: string) =>
    ["apiKeyGameShares", shareId] as const,
  apiKeyStatus: (gameId: string) => ["apiKeyStatus", gameId] as const,

  // Platforms
  platforms: ["platforms"] as const,
  genres: ["genres"] as const,
  themes: ["themes"] as const,

  // Games
  games: ["games"] as const,
  gameTemplate: ["games", "template"] as const,
  /** Every game list; under `games`, so invalidating games refreshes them. */
  gameLists: ["games", "list"] as const,
  gameList: (params: Record<string, unknown>) =>
    ["games", "list", params] as const,
  game: (id: string) => ["games", id] as const,
  gameWithParams: (params: {
    search?: string;
    sortBy?: string;
    sortDir?: string;
    filter?: string;
  }) => ["games", params] as const,
  favoriteGames: ["games", "favorites"] as const,

  // Sessions
  gameSessions: ["gameSessions"] as const,
  gameSessionsForGame: (gameId: string) => ["gameSessions", gameId] as const,
  userSessions: ["userSessions"] as const,
  userSessionsWithParams: (params: { search?: string; sortBy?: string }) =>
    ["userSessions", params] as const,
  /** What a share link offers now: its game and plays left. */
  shareInfo: (token: string) => ["shareInfo", token] as const,

  // Users
  users: ["users"] as const,
  user: (id: string) => ["users", id] as const,
  currentUser: ["currentUser"] as const,
  currentUserStats: ["currentUser", "stats"] as const,
  backendUser: ["backend-user"] as const,
  adminUsers: ["admin-users"] as const,

  // Roles
  roles: ["roles"] as const,

  // Institutions/Organizations
  institutions: ["institutions"] as const,
  institution: (id: string) => ["institution", id] as const,
  institutionMembersBase: ["institution-members"] as const,
  institutionMembers: (institutionId: string) =>
    ["institution-members", institutionId] as const,
  verifiedPages: (institutionId: string) => ["verified-pages", institutionId] as const,
  allVerifiedPages: ["verified-pages", "all"] as const,
  gameEmbed: (gameId: string) => ["game-embed", gameId] as const,

  // Organisations registering themselves
  declaration: ["declaration"] as const,
  organisationConsent: (institutionId: string) =>
    ["organisation-consent", institutionId] as const,
  /** Under `institutions`, so changing institutions refreshes it. */
  recentlyRegisteredInstitutions: ["institutions", "recently-registered"] as const,

  // Invites
  invites: ["invites"] as const,
  institutionInvitesBase: ["institution-invites"] as const,
  institutionInvites: (institutionId: string) =>
    ["institution-invites", institutionId] as const,
  allInvites: ["all-invites"] as const,

  // Notifications: the caller's bell
  notifications: ["notifications"] as const,

  // Workshops
  workshops: ["workshops"] as const,
  workshopsByInstitution: (institutionId: string) =>
    ["workshops", institutionId] as const,
  workshop: (id: string) => ["workshop", id] as const,
  guestToken: (guestId: string) => ["guestToken", guestId] as const,
  publicWorkshop: (slug: string) => ["publicWorkshop", slug] as const,

  // System
  version: ["version"] as const,
  serverStatus: ["serverStatus"] as const,
  systemSettings: ["systemSettings"] as const,
  site: ["site"] as const,
  engineModels: ["engineModels"] as const,
  updates: ["updates"] as const,
  systemConstraints: ["systemConstraints"] as const,

  // Translations
  translations: (language: string, namespace: string) =>
    ["translations", language, namespace] as const,
} as const;
