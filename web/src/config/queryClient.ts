import { QueryClient, QueryCache, MutationCache } from '@tanstack/react-query';
import { getBaseUrl } from '@/common/lib/url';
import { notifications } from '@mantine/notifications';
import { apiLogger } from './logger';
import i18n from '../i18n';
import type { HttpxErrorResponse } from '../api/generated';

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60 * 5, // 5 minutes
      retry: (failureCount, error: Error & { status?: number; type?: string }) => {
        // Log retry attempt
        apiLogger.warning('Query retry attempt', {
          failureCount,
          error: {
            message: error?.message,
            status: error?.status,
            type: error?.type || 'Unknown'
          }
        });

        // Don't retry on 4xx errors (client errors)
        if (error && typeof error === 'object' && 'status' in error) {
          const status = error.status;
          if (status && status >= 400 && status < 500) {
            apiLogger.debug('Skipping retry for client error', { status, errorType: error?.type });
            return false;
          }
        }
        // Retry up to 1 time for other errors
        const shouldRetry = failureCount < 1;
        apiLogger.debug('Retry decision', { failureCount, shouldRetry });
        return shouldRetry;
      },
    },
    mutations: {
      retry: (failureCount, error: Error & { status?: number; type?: string }) => {
        // Log retry attempt
        apiLogger.warning('Mutation retry attempt', {
          failureCount,
          error: {
            message: error?.message,
            status: error?.status,
            type: error?.type || 'Unknown'
          }
        });

        // Don't retry on 4xx errors (client errors)
        if (error && typeof error === 'object' && 'status' in error) {
          const status = error.status;
          if (status && status >= 400 && status < 500) {
            apiLogger.debug('Skipping retry for client error', { status, errorType: error?.type });
            return false;
          }
        }
        // Retry up to 1 time for other errors
        const shouldRetry = failureCount < 1;
        apiLogger.debug('Retry decision', { failureCount, shouldRetry });
        return shouldRetry;
      },
    },
  },
});

// Global error handler function
export function handleApiError(error: HttpxErrorResponse | Error | { status?: number; type?: string; code?: string; message?: string }) {
  // Extract status code from the error
  let status = 0;
  let message = 'An unexpected error occurred';
  let errorType = 'Unknown';
  let errorCode = '';
  let errorDetails: Record<string, unknown> = {};
  
  if (error && typeof error === 'object') {
    // Handle fetch errors or HTTP errors
    if ('status' in error) {
      status = (error as { status?: number }).status || 0;
      errorType = 'HTTP Error';
    }
    
    // Handle structured API errors
    if ('message' in error && typeof error.message === 'string') {
      message = error.message;
    }
    
    // Handle error type from API response
    if ('type' in error && typeof error.type === 'string') {
      errorType = error.type;
    }
    
    // Handle error code from API response (new structured errors)
    if ('code' in error && typeof error.code === 'string') {
      errorCode = error.code;
      errorType = error.code; // Use code as type if available
    }
    
    // Handle network errors
    if (error instanceof TypeError && error.message.includes('fetch')) {
      message = 'Network error. Please check your connection.';
      status = 0;
      errorType = 'Network Error';
    }

    // Collect additional error details for logging
    const errorObj = error as Record<string, unknown>;
    errorDetails = {
      status,
      message: errorObj.message || message,
      type: errorObj.type || errorType,
      code: errorObj.code || errorCode,
      stack: errorObj.stack,
      name: errorObj.name,
      // Include any other error properties
      ...Object.fromEntries(
        Object.entries(errorObj).filter(([key]) => 
          !['status', 'message', 'type', 'code', 'stack', 'name'].includes(key)
        )
      )
    };
  }

  // Log the complete error with details
  apiLogger.error('API Error occurred', {
    errorType,
    errorCode,
    status,
    message,
    timestamp: new Date().toISOString(),
    userAgent: typeof window !== 'undefined' ? window.navigator.userAgent : 'Server-side',
    url: typeof window !== 'undefined' ? window.location.href : 'Server-side',
    errorDetails
  });

  // Helper to get error translations from the 'errors' namespace
  const te = (key: string, options?: object) => i18n.t(key, { ns: 'errors', ...options });

  // Handle specific error codes
  if (errorCode === 'invalid_platform') {
    notifications.show({
      title: te('titles.validation'),
      message: i18n.t('apiKeys.errors.invalidPlatform', { ns: 'common', defaultValue: message }),
      color: 'orange',
    });
    return;
  }
  
  if (errorCode === 'validation_error') {
    notifications.show({
      title: te('titles.validation'),
      message: message || te('validation'),
      color: 'orange',
    });
    return;
  }

  // Handle workshop inactive error - participant's workshop has been deactivated
  if (errorCode === 'auth_workshop_inactive') {
    notifications.show({
      title: te('titles.workshopInactive'),
      message: te('workshopInactive'),
      color: 'orange',
    });
    // Clear session and redirect to a workshop inactive page or home
    if (typeof window !== 'undefined') {
      // Dispatch custom event for auth provider to handle
      window.dispatchEvent(new CustomEvent('cgl:workshop_inactive'));
    }
    return;
  }

  // Handle invalid participant token
  if (errorCode === 'auth_invalid_token') {
    notifications.show({
      title: te('titles.authentication'),
      message: te('invalidToken'),
      color: 'red',
    });
    return;
  }

  // Show appropriate notification based on status code (fallback if no error code handled)
  switch (status) {
    case 401:
      notifications.show({
        title: te('titles.authentication'),
        message: te('unauthorized'),
        color: 'red',
      });
      // Redirect to login page
      if (typeof window !== 'undefined' && !window.location.pathname.startsWith('/auth/')) {
        window.location.href = `${getBaseUrl()}/auth/login`;
      }
      break;
    case 403:
      notifications.show({
        title: te('titles.permission'),
        message: te('forbidden'),
        color: 'red',
      });
      break;
    case 404:
      notifications.show({
        title: te('titles.notFound'),
        message: te('notFound'),
        color: 'orange',
      });
      break;
    case 422:
      notifications.show({
        title: te('titles.validation'),
        message: message || te('validation'),
        color: 'orange',
      });
      break;
    case 500:
    case 502:
    case 503:
    case 504:
      notifications.show({
        title: te('titles.server'),
        message: te('server'),
        color: 'red',
      });
      break;
    case 0:
      notifications.show({
        title: te('titles.network'),
        message: te('network'),
        color: 'red',
      });
      break;
    default:
      // For other errors, show a generic error message
      if (status >= 400 && status < 500) {
        notifications.show({
          title: te('titles.error'),
          message: message || te('generic'),
          color: 'orange',
        });
      } else {
        notifications.show({
          title: te('titles.error'),
          message: message || te('unexpected'),
          color: 'red',
        });
      }
      break;
  }
}
