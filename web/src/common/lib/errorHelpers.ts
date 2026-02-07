import i18n from "../../i18n";
import {
  ErrorCodes,
  extractErrorCode,
  extractRawErrorCode,
  isKnownErrorCode,
  type ErrorCode,
} from "../types/errorCodes";

/**
 * Mapping from error codes to i18n keys.
 */
const ERROR_CODE_I18N_MAP: Record<
  ErrorCode,
  { titleKey: string; messageKey: string }
> = {
  [ErrorCodes.GENERIC]: {
    titleKey: "errors:titles.error",
    messageKey: "errors:generic",
  },
  [ErrorCodes.VALIDATION]: {
    titleKey: "errors:titles.validation",
    messageKey: "errors:validation",
  },
  [ErrorCodes.UNAUTHORIZED]: {
    titleKey: "errors:titles.authentication",
    messageKey: "errors:unauthorized",
  },
  [ErrorCodes.FORBIDDEN]: {
    titleKey: "errors:titles.permission",
    messageKey: "errors:forbidden",
  },
  [ErrorCodes.NOT_FOUND]: {
    titleKey: "errors:titles.notFound",
    messageKey: "errors:notFound",
  },
  [ErrorCodes.CONFLICT]: {
    titleKey: "errors:titles.error",
    messageKey: "errors:conflict",
  },
  [ErrorCodes.INVALID_PLATFORM]: {
    titleKey: "errors:titles.validation",
    messageKey: "errors:invalidPlatform",
  },
  [ErrorCodes.INVALID_INPUT]: {
    titleKey: "errors:titles.validation",
    messageKey: "errors:validation",
  },
  [ErrorCodes.SERVER_ERROR]: {
    titleKey: "errors:titles.server",
    messageKey: "errors:server",
  },
  [ErrorCodes.USER_NOT_REGISTERED]: {
    titleKey: "errors:titles.authentication",
    messageKey: "errors:userNotRegistered",
  },
  [ErrorCodes.INVALID_API_KEY]: {
    titleKey: "errors:titles.authentication",
    messageKey: "errors:invalidApiKey",
  },
  [ErrorCodes.ORG_VERIFICATION_REQUIRED]: {
    titleKey: "errors:titles.authentication",
    messageKey: "errors:orgVerificationRequired",
  },
  [ErrorCodes.BILLING_NOT_ACTIVE]: {
    titleKey: "errors:titles.billing",
    messageKey: "errors:billingNotActive",
  },
  [ErrorCodes.RATE_LIMIT_EXCEEDED]: {
    titleKey: "errors:titles.aiError",
    messageKey: "errors:rateLimitExceeded",
  },
  [ErrorCodes.INSUFFICIENT_QUOTA]: {
    titleKey: "errors:titles.billing",
    messageKey: "errors:insufficientQuota",
  },
  [ErrorCodes.CONTENT_FILTERED]: {
    titleKey: "errors:titles.aiError",
    messageKey: "errors:contentFiltered",
  },
  [ErrorCodes.DUPLICATE_NAME]: {
    titleKey: "errors:titles.error",
    messageKey: "errors:duplicateName",
  },
  [ErrorCodes.AI_ERROR]: {
    titleKey: "errors:titles.aiError",
    messageKey: "errors:aiError",
  },
  [ErrorCodes.AUTH_WORKSHOP_INACTIVE]: {
    titleKey: "errors:titles.authentication",
    messageKey: "errors:workshopInactive",
  },
};

export interface TranslatedError {
  title: string;
  message: string;
  color: "red" | "orange";
  code: string | null;
  isKnown: boolean;
}

/**
 * Get the color for an error code.
 */
function getErrorColor(errorCode: string | null): "red" | "orange" {
  if (!errorCode) return "red";

  switch (errorCode) {
    case ErrorCodes.VALIDATION:
    case ErrorCodes.INVALID_INPUT:
    case ErrorCodes.NOT_FOUND:
    case ErrorCodes.CONFLICT:
    case ErrorCodes.INVALID_PLATFORM:
    case ErrorCodes.DUPLICATE_NAME:
      return "orange";
    default:
      return "red";
  }
}

/**
 * Translate an error code to user-friendly text.
 * Pass an error code string directly.
 */
export function translateErrorCode(code: string): TranslatedError {
  const isKnown = isKnownErrorCode(code);

  if (isKnown) {
    const keys = ERROR_CODE_I18N_MAP[code];
    return {
      title: i18n.t(keys.titleKey),
      message: i18n.t(keys.messageKey),
      color: getErrorColor(code),
      code,
      isKnown: true,
    };
  }

  // Unknown error code - show generic message with code hint
  return {
    title: i18n.t("errors:titles.error"),
    message: `${i18n.t("errors:generic")} (${code})`,
    color: "red",
    code,
    isKnown: false,
  };
}

/**
 * Translate an API error object to user-friendly text.
 * Automatically extracts the error code from the error object.
 */
export function translateError(error: unknown): TranslatedError {
  const knownCode = extractErrorCode(error);
  const rawCode = extractRawErrorCode(error);

  if (knownCode) {
    return translateErrorCode(knownCode);
  }

  if (rawCode) {
    // Unknown code - show generic with hint
    return {
      title: i18n.t("errors:titles.error"),
      message: `${i18n.t("errors:generic")} (${rawCode})`,
      color: "red",
      code: rawCode,
      isKnown: false,
    };
  }

  // No error code at all
  return {
    title: i18n.t("errors:titles.error"),
    message: i18n.t("errors:generic"),
    color: "red",
    code: null,
    isKnown: false,
  };
}

/**
 * Get translated error for a known ErrorCode constant.
 */
export function getErrorText(code: ErrorCode): {
  title: string;
  message: string;
} {
  const keys = ERROR_CODE_I18N_MAP[code];
  return {
    title: i18n.t(keys.titleKey),
    message: i18n.t(keys.messageKey),
  };
}
