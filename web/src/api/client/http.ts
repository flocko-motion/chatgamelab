const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';

export function getApiConfig() {
  return {
    baseUrl: API_BASE_URL,
    baseApiParams: {
      headers: {
        'Content-Type': 'application/json',
      },
    },
  };
}
