import { useSyncExternalStore } from "react";
import { ErrorModal } from "./ErrorModal";
import {
  subscribeErrorModal,
  getErrorModalState,
  dismissErrorModal,
} from "../lib/globalErrorModal";

/**
 * Renders the global error modal. Mount once at the app root.
 *
 * Errors are shown via the imperative `showErrorModal()` API from
 * `@/common/lib/globalErrorModal` - no props needed.
 */
export function GlobalErrorModal() {
  const modalState = useSyncExternalStore(
    subscribeErrorModal,
    getErrorModalState,
  );

  return (
    <ErrorModal
      opened={modalState !== null}
      onClose={dismissErrorModal}
      errorCode={modalState?.code ?? undefined}
      title={modalState?.title}
      message={modalState?.message}
    />
  );
}
