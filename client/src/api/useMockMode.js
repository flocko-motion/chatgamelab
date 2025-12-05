// Centralized mock mode detection - single source of truth
// Checks URL parameter once and provides consistent mock mode state across the app

let mockModeCache = null;

export const isMockMode = () => {
  // Cache the result so we only check URL parameter once
  if (mockModeCache === null) {
    mockModeCache = new URLSearchParams(window.location.search).get('mock') === 'true';
    
    if (mockModeCache) {
      console.log('[MOCK MODE] Mock mode enabled - detected ?mock=true in URL');
    }
  }
  
  return mockModeCache;
};

// React hook version for components that need reactive updates
export const useMockMode = () => {
  return isMockMode();
};