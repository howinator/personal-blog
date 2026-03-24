FROM --platform=$BUILDPLATFORM node:22-alpine AS deps
WORKDIR /src
COPY package.json pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile --prod

FROM --platform=$BUILDPLATFORM hugomods/hugo:0.147.0 AS build
WORKDIR /src
COPY site/ .
COPY --from=deps /src/node_modules /src/node_modules
RUN hugo --minify

FROM --platform=$TARGETPLATFORM nginx:alpine
COPY --from=build /src/public /usr/share/nginx/html
EXPOSE 80
