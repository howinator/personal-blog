FROM --platform=$BUILDPLATFORM hugomods/hugo:0.147.0 AS build
WORKDIR /src
COPY site/ .
RUN hugo --minify

FROM --platform=$TARGETPLATFORM nginx:alpine
COPY --from=build /src/public /usr/share/nginx/html
EXPOSE 80
