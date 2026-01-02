# TanStack Query Integration

This document explains how to use the TanStack Query hooks that have been configured for the generated API.

## Overview

The TanStack Query hooks provide:
- Automatic caching and background refetching
- Loading and error state management
- Optimistic updates and cache invalidation
- Type-safe API calls based on the OpenAPI spec
- **Global error handling with automatic toast notifications**

## Available Hooks

### API Keys
- `useApiKeys()` - List all API key shares
- `useCreateApiKey()` - Create a new API key
- `useUpdateApiKey()` - Update/share an API key

### Games
- `useGames()` - List all games
- `useGame(id)` - Get a specific game by ID
- `useCreateGame()` - Create a new game
- `useUpdateGame()` - Update an existing game

### Game Sessions
- `useGameSessions(gameId)` - List sessions for a specific game
- `useCreateGameSession()` - Create a new game session

### Users
- `useUsers()` - List all users
- `useCurrentUser()` - Get the currently authenticated user
- `useUser(id)` - Get a specific user by ID
- `useUpdateUser()` - Update a user
- `useCreateUser()` - Create a new user (dev only)

### System
- `useVersion()` - Get server version info

## Error Handling

The integration includes **automatic global error handling** that shows toast notifications for API errors **and logs detailed error information**:

### Error Types Handled

- **401 Unauthorized** - Shows authentication error with red notification
- **403 Forbidden** - Shows permission denied error with red notification  
- **404 Not Found** - Shows not found error with orange notification
- **422 Validation Error** - Shows validation error with orange notification
- **500/502/503/504 Server Errors** - Shows server error with red notification
- **Network Errors** - Shows network error with red notification
- **Other 4xx Errors** - Shows generic error with orange notification
- **Other 5xx Errors** - Shows generic error with red notification

### Detailed Logging

All API errors are automatically logged with comprehensive information:

```typescript
// Example of logged error data
{
  errorType: "HTTP Error",
  status: 404,
  message: "Game not found",
  timestamp: "2025-01-02T20:43:00.000Z",
  userAgent: "Mozilla/5.0...",
  url: "http://localhost:5173/games/123",
  errorDetails: {
    status: 404,
    message: "Game not found",
    type: "NotFound",
    stack: "...",
    name: "Error",
    // Additional error properties
  }
}
```

### Retry Logic & Logging

- **4xx errors** (client errors): No retry (user action required) - logged as debug
- **5xx errors** (server errors): Retry up to 1 time - each retry attempt is logged
- **Network errors**: Retry up to 1 time - retry attempts logged with warning level

Retry attempts are logged with:
- Failure count
- Error details (status, message, type)
- Retry decision (whether retry will be attempted)

### Usage Examples

#### Basic Query with Error Handling
```typescript
import { useGames } from '../../../api/client';

function GameList() {
  const { data: games, isLoading, error } = useGames();

  if (isLoading) return <div>Loading...</div>;
  if (error) {
    // Error is automatically shown as toast notification
    // You can still handle it locally if needed
    return <div>Error: {error.message}</div>;
  }

  return (
    <div>
      {games?.map(game => (
        <div key={game.id}>{game.name}</div>
      ))}
    </div>
  );
}
```

#### Mutation with Automatic Error Handling
```typescript
import { useCreateGame } from '../../../api/client';

function CreateGameForm() {
  const createGameMutation = useCreateGame();

  const handleSubmit = (name: string) => {
    createGameMutation.mutate(
      { name },
      {
        onSuccess: () => {
          // Success handling
          console.log('Game created successfully');
        },
        // No need for onError - errors are handled globally!
      }
    );
  };

  return (
    <button 
      onClick={() => handleSubmit('My Game')}
      disabled={createGameMutation.isPending}
    >
      {createGameMutation.isPending ? 'Creating...' : 'Create Game'}
    </button>
  );
}
```

#### Manual Error Handling (Optional)
If you need to handle errors locally in addition to the global notifications:

```typescript
import { useCreateGame, handleApiError } from '../../../api/client';

function CreateGameForm() {
  const createGameMutation = useCreateGame();

  const handleSubmit = (name: string) => {
    createGameMutation.mutate(
      { name },
      {
        onSuccess: () => {
          console.log('Game created successfully');
        },
        onError: (error) => {
          // Global notification already shown
          // Add any additional local handling if needed
          console.error('Local error handling:', error);
          // Or call the global handler manually
          handleApiError(error);
        },
      }
    );
  };

  // ... rest of component
}
```

## Query Keys

The hooks use structured query keys for cache management:

```typescript
export const queryKeys = {
  apiKeys: ['apiKeys'] as const,
  games: ['games'] as const,
  gameSessions: ['gameSessions'] as const,
  users: ['users'] as const,
  currentUser: ['currentUser'] as const,
  version: ['version'] as const,
};
```

You can use these keys for manual cache operations:

```typescript
import { useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '../../../api/client';

function SomeComponent() {
  const queryClient = useQueryClient();

  const invalidateGamesCache = () => {
    queryClient.invalidateQueries({ queryKey: queryKeys.games });
  };

  const prefetchGame = (id: string) => {
    queryClient.prefetchQuery({
      queryKey: [...queryKeys.games, id],
      queryFn: () => apiClient.games.gamesDetail(id).then(r => r.data),
    });
  };
}
```

## Development Notes

- The hooks are automatically generated from the OpenAPI spec
- All API calls are type-safe
- Cache invalidation is handled automatically for mutations
- Loading states are provided by default
- **Error notifications are shown automatically for all API errors**
- Error handling follows the OpenAPI error response structure

## Setup Requirements

To use the error handling, ensure you have:

1. **@mantine/notifications** installed (already done)
2. **Notifications provider** in your app (configured in `AppProviders.tsx`)
3. **CSS imports** for notifications (configured in `main.tsx`)

The setup is already complete and ready to use!

## Next Steps

To use these hooks in your components:

1. Import the hooks you need from `../../../api/client`
2. Use them in your React components
3. Handle loading states as needed
4. **Errors will automatically show as toast notifications**
5. The cache will be managed automatically by TanStack Query
