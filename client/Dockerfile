# Build React frontend
FROM node:20 AS build

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm install

# Copy git directory for version info
COPY .git .git
COPY . .

# Generate version file with git commit and datetime
RUN GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown") && \
    BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ") && \
    echo "export const version = \"${GIT_COMMIT}\";" > ./src/version.js && \
    echo "export const buildTime = \"${BUILD_TIME}\";" >> ./src/version.js

RUN npm run build

# Runtime image: use Node + express server.js to serve the built app
FROM node:20-alpine AS runtime

WORKDIR /app

# Only need minimal deps for server.js
COPY package.json package-lock.json ./
RUN npm install --only=production

# Copy compiled build and server entrypoint
COPY --from=build /app/build /app/build
COPY server.js /app/server.js

ENV SERVER_PORT=3000
EXPOSE 3000

CMD ["node", "server.js"]
