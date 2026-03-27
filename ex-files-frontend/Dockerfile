FROM oven/bun:1 AS build

WORKDIR /src
COPY package.json bun.lock ./
RUN bun install --frozen-lockfile
COPY . .
RUN bun run build

FROM node:22-alpine
WORKDIR /app
COPY --from=build /src/build ./build
COPY --from=build /src/package.json .
COPY --from=build /src/node_modules ./node_modules
EXPOSE 3000
ENV PORT=3000
CMD ["node", "build"]
