export { getApiConfig, createAuthenticatedApiConfig } from './http';

export * from '../generated';

import { Api } from '../generated';
import { getApiConfig } from './http';

export const apiClient = new Api(getApiConfig());

export * from '../hooks';
export { handleApiError } from '../../config/queryClient';
