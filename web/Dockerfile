# Build stage for React frontend
FROM node:24-alpine AS build

WORKDIR /app

# Copy package files first for better caching
COPY package*.json ./
RUN npm ci

# Copy source code
COPY . .

# Generate API types first
RUN npm run gen:api

# Build the application with version info
ARG VERSION=dev
ARG BUILD_TIME=unknown
RUN echo "export const version = \"${VERSION}\";" > src/version.js && \
    echo "export const buildTime = \"${BUILD_TIME}\";" >> src/version.js && \
    npm run build

# Production stage - serve static files with nginx
FROM nginx:1.27-alpine

# Copy nginx configuration
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Copy entrypoint script for runtime config generation
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Copy built app to nginx serve directory
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 80

ENTRYPOINT ["/entrypoint.sh"]
CMD ["nginx", "-g", "daemon off;"]
