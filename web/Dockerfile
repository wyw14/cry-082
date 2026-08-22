# syntax=docker/dockerfile:1.7
FROM node:22-alpine AS console-build
WORKDIR /console
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci
COPY . .
RUN npm run typecheck && npm run build

FROM nginx:1.27-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=console-build /console/dist /usr/share/nginx/html
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=3s --retries=6 \
    CMD wget -q -O - http://127.0.0.1:8080/console-health || exit 1
