# Prepare files and a temp image to build.
FROM node:16-alpine3.11 AS builder
COPY package.json dappetizer.config.ts tsconfig.json /app/
COPY src /app/src

# Build the app.
WORKDIR /app
RUN npm install && npm run build

# Publish files to a final image.
FROM node:16-alpine3.11
COPY --from=builder /app/package.json /app/dappetizer.config.ts /app/
COPY --from=builder /app/dist /app/dist
COPY --from=builder /app/node_modules /app/node_modules

# Run the app.
WORKDIR /app
CMD ["./node_modules/.bin/dappetizer", "start"]
