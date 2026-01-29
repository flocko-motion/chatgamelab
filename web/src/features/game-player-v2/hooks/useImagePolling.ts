import { useState, useEffect, useRef, useCallback } from 'react';
import { config } from '@/config/env';

interface ImageStatus {
  hash: string;
  isComplete: boolean;
  hasError?: boolean;
  errorCode?: string;
  errorMsg?: string;
  exists: boolean;
  isOrganisationUnverified?: boolean;
}

interface UseImagePollingOptions {
  messageId: string;
  enabled?: boolean;
  pollingInterval?: number;
}

interface UseImagePollingResult {
  imageUrl: string | null;
  isLoading: boolean;
  isComplete: boolean;
  hasError: boolean;
  errorCode: string | null;
  isOrganisationUnverified: boolean;
}

const DEFAULT_POLLING_INTERVAL = 2000; // 2 seconds
const MAX_STATUS_RETRIES = 3;

export function useImagePolling({
  messageId,
  enabled = true,
  pollingInterval = DEFAULT_POLLING_INTERVAL,
}: UseImagePollingOptions): UseImagePollingResult {
  const [imageUrl, setImageUrl] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isComplete, setIsComplete] = useState(false);
  const [hasError, setHasError] = useState(false);
  const [errorCode, setErrorCode] = useState<string | null>(null);
  const [isOrganisationUnverified, setIsOrganisationUnverified] = useState(false);
  const lastHashRef = useRef<string | null>(null);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const retryCountRef = useRef(0);

  const checkStatus = useCallback(async () => {
    if (!messageId || !enabled) return;

    try {
      const response = await fetch(
        `${config.API_BASE_URL}/messages/${messageId}/image/status`
      );

      if (!response.ok) {
        return;
      }

      const status: ImageStatus = await response.json();

      if (!status.exists) {
        // No image yet, keep polling
        return;
      }

      // Check if hash changed (new image available)
      if (status.hash !== lastHashRef.current) {
        lastHashRef.current = status.hash;
        // Add cache-busting param to force reload
        const cacheBuster = Date.now();
        setImageUrl(
          `${config.API_BASE_URL}/messages/${messageId}/image?v=${cacheBuster}`
        );
        setIsLoading(false);
      }

      // Stop polling on complete or error
      if (status.isComplete || status.hasError) {
        if (status.isComplete) setIsComplete(true);
        if (status.hasError) {
          setHasError(true);
          setErrorCode(status.errorCode || null);
          if (status.isOrganisationUnverified) {
            setIsOrganisationUnverified(true);
          }
        }
        if (intervalRef.current) {
          clearInterval(intervalRef.current);
          intervalRef.current = null;
        }
      }
    } catch (error) {
      retryCountRef.current++;
      console.error('Failed to check image status:', error, `(attempt ${retryCountRef.current}/${MAX_STATUS_RETRIES})`);
      
      // Stop polling after max retries
      if (retryCountRef.current >= MAX_STATUS_RETRIES) {
        setHasError(true);
        setErrorCode('network_error');
        if (intervalRef.current) {
          clearInterval(intervalRef.current);
          intervalRef.current = null;
        }
      }
    }
  }, [messageId, enabled]);

  useEffect(() => {
    if (!enabled || !messageId) {
      return;
    }

    // Reset state for new message
    setIsLoading(true);
    setIsComplete(false);
    setHasError(false);
    setErrorCode(null);
    setIsOrganisationUnverified(false);
    setImageUrl(null);
    lastHashRef.current = null;
    retryCountRef.current = 0;

    // Initial check
    checkStatus();

    // Start polling
    intervalRef.current = setInterval(checkStatus, pollingInterval);

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    };
  }, [messageId, enabled, pollingInterval, checkStatus]);

  return {
    imageUrl,
    isLoading,
    isComplete,
    hasError,
    errorCode,
    isOrganisationUnverified,
  };
}
