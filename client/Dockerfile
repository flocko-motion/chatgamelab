# Build React frontend
FROM node:20 AS build

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm install

COPY . .

# Generate version file (mimic legacy Dockerfile behaviour)
RUN echo "export const version =\"$(date +%y%m%d).$(printf '%04d' $(date +%s) | tail -c 4)\"" > ./src/version.js

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
