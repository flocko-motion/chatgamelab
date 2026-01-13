// This file is overwritten by the CI/CD pipeline during Docker builds
// Local development version - will be replaced with actual release version in production
export const version = "dev-" + new Date().getTime().toString(36);
export const buildTime = new Date().toISOString();
