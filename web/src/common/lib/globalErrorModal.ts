/**
 * Global error modal system.
 *
 * Provides an imperative API to show error modals from anywhere in the app,
 * including non-React code (e.g. handleApiError in queryClient.ts).
 *
 * Usage:
 *   showErrorModal({ code: 'ai_error' })
 *   showErrorModal({ code: 'duplicate_name', onDismiss: () => navigate('/games') })
 *   showErrorModal({ title: 'Custom', message: 'Something happened' })
 */

export interface GlobalErrorModalOptions {
  /** Machine-readable error code - the modal translates it via i18n */
  code?: string | null;
  /** Override title (bypasses i18n lookup) */
  title?: string;
  /** Override message (bypasses i18n lookup) */
  message?: string;
  /** Called when the user dismisses the modal (close button, backdrop click, etc.) */
  onDismiss?: () => void;
}

type Subscriber = (options: GlobalErrorModalOptions | null) => void;

let current: GlobalErrorModalOptions | null = null;
const subscribers = new Set<Subscriber>();

function notify() {
  subscribers.forEach((fn) => fn(current));
}

/** Show a global error modal. Only one modal at a time.
 *  Skips notification if an identical error is already displayed (dedup). */
export function showErrorModal(options: GlobalErrorModalOptions) {
  // Deduplicate: if the same error code (or same message when no code) is
  // already showing, skip the update to avoid cascading re-renders when
  // multiple queries fail simultaneously (e.g. backend down).
  if (current) {
    const sameCode = options.code && current.code === options.code;
    const sameMessage =
      !options.code && !current.code && options.message === current.message;
    if (sameCode || sameMessage) {
      return;
    }
  }
  current = options;
  notify();
}

/** Dismiss the current global error modal. */
export function dismissErrorModal() {
  const onDismiss = current?.onDismiss;
  current = null;
  notify();
  onDismiss?.();
}

/** Subscribe to modal state changes. Returns an unsubscribe function. */
export function subscribeErrorModal(fn: Subscriber): () => void {
  subscribers.add(fn);
  return () => subscribers.delete(fn);
}

/** Get the current modal state (for initial render). */
export function getErrorModalState(): GlobalErrorModalOptions | null {
  return current;
}
