import { QueryClient } from "@tanstack/react-query";
import { getBaseUrl } from "@/common/lib/url";
import { showErrorModal } from "@/common/lib/globalErrorModal";
import { apiLogger } from "./logger";
import type { HttpxErrorResponse } from "../api/generated";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60 * 5, // 5 minutes
      retry: (
        failureCount,
        error: Error & { status?: number; type?: string },
      ) => {
        // Log retry attempt
        apiLogger.warning("Query retry attempt", {
          failureCount,
          error: {
            message: error?.message,
            status: error?.status,
            type: error?.type || "Unknown",
          },
        });

        // Don't retry on 4xx errors (client errors)
        if (error && typeof error === "object" && "status" in error) {
          const status = error.status;
          if (status && status >= 400 && status < 500) {
            apiLogger.debug("Skipping retry for client error", {
              status,
              errorType: error?.type,
            });
            return false;
          }
        }
        // Retry up to 1 time for other errors
        const shouldRetry = failureCount < 1;
        apiLogger.debug("Retry decision", { failureCount, shouldRetry });
        return shouldRetry;
      },
    },
    mutations: {
      retry: (
        failureCount,
        error: Error & { status?: number; type?: string },
      ) => {
        // Log retry attempt
        apiLogger.warning("Mutation retry attempt", {
          failureCount,
          error: {
            message: error?.message,
            status: error?.status,
            type: error?.type || "Unknown",
          },
        });

        // Don't retry on 4xx errors (client errors)
        if (error && typeof error === "object" && "status" in error) {
          const status = error.status;
          if (status && status >= 400 && status < 500) {
            apiLogger.debug("Skipping retry for client error", {
              status,
              errorType: error?.type,
            });
            return false;
          }
        }
        // Retry up to 1 time for other errors
        const shouldRetry = failureCount < 1;
        apiLogger.debug("Retry decision", { failureCount, shouldRetry });
        return shouldRetry;
      },
    },
  },
});

// Global error handler function
export function handleApiError(
  error:
    | HttpxErrorResponse
    | Error
    | { status?: number; type?: string; code?: string; message?: string },
) {
  // Extract status code from the error
  let status = 0;
  let message = "An unexpected error occurred";
  let errorType = "Unknown";
  let errorCode = "";
  let errorDetails: Record<string, unknown> = {};

  if (error && typeof error === "object") {
    // Handle fetch errors or HTTP errors
    if ("status" in error) {
      status = (error as { status?: number }).status || 0;
      errorType = "HTTP Error";
    }

    // The generated API client throws the HttpResponse object on error.
    // The parsed error body is in the `.error` property — extract it first.
    const errorObj = error as Record<string, unknown>;
    const nested =
      errorObj.error && typeof errorObj.error === "object"
        ? (errorObj.error as Record<string, unknown>)
        : null;

    // Handle structured API errors (prefer nested body, fall back to top-level)
    const rawMessage = nested?.message ?? errorObj.message;
    if (typeof rawMessage === "string") {
      message = rawMessage;
    }

    // Handle error type from API response
    const rawType = nested?.type ?? errorObj.type;
    if (typeof rawType === "string") {
      errorType = rawType;
    }

    // Handle error code from API response (new structured errors)
    const rawCode = nested?.code ?? errorObj.code;
    if (typeof rawCode === "string" && rawCode) {
      errorCode = rawCode;
      errorType = rawCode; // Use code as type if available
    }

    // Handle network errors
    if (error instanceof TypeError && error.message.includes("fetch")) {
      message = "Network error. Please check your connection.";
      status = 0;
      errorType = "Network Error";
    }

    // Collect additional error details for logging
    errorDetails = {
      status,
      message: errorObj.message || message,
      type: errorObj.type || errorType,
      code: errorObj.code || errorCode,
      stack: errorObj.stack,
      name: errorObj.name,
      // Include any other error properties
      ...Object.fromEntries(
        Object.entries(errorObj).filter(
          ([key]) =>
            !["status", "message", "type", "code", "stack", "name"].includes(
              key,
            ),
        ),
      ),
    };
  }

  // Log the complete error with details
  apiLogger.error("API Error occurred", {
    errorType,
    errorCode,
    status,
    message,
    timestamp: new Date().toISOString(),
    userAgent:
      typeof window !== "undefined"
        ? window.navigator.userAgent
        : "Server-side",
    url: typeof window !== "undefined" ? window.location.href : "Server-side",
    errorDetails,
  });

  // If we have a known error code, the ErrorModal handles translation automatically.
  // Special cases get onDismiss for side effects.
  if (errorCode) {
    // Workshop inactive: dispatch custom event on dismiss
    if (errorCode === "auth_workshop_inactive") {
      showErrorModal({
        code: errorCode,
        onDismiss: () => {
          if (typeof window !== "undefined") {
            window.dispatchEvent(new CustomEvent("cgl:workshop_inactive"));
          }
        },
      });
      return;
    }

    // All other known error codes — just show the modal
    showErrorModal({ code: errorCode });
    return;
  }

  // Fallback: no error code — map HTTP status to an error code
  const statusCodeMap: Record<number, string> = {
    401: "unauthorized",
    403: "forbidden",
    404: "not_found",
    422: "validation_error",
    500: "server_error",
    502: "server_error",
    503: "server_error",
    504: "server_error",
  };

  const mappedCode = statusCodeMap[status];

  if (status === 401) {
    showErrorModal({
      code: "unauthorized",
      onDismiss: () => {
        if (
          typeof window !== "undefined" &&
          !window.location.pathname.startsWith("/auth/")
        ) {
          window.location.href = `${getBaseUrl()}/auth/login`;
        }
      },
    });
    return;
  }

  if (mappedCode) {
    showErrorModal({ code: mappedCode });
    return;
  }

  // Network error (status 0)
  if (status === 0) {
    showErrorModal({ code: "error", message });
    return;
  }

  // Generic fallback
  showErrorModal({ code: "error", message });
}
