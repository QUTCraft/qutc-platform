FROM node:22-alpine AS build
WORKDIR /app
RUN corepack enable
ARG NPM_REGISTRY=https://registry.npmjs.org/
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN pnpm install --registry=$NPM_REGISTRY --frozen-lockfile
COPY . .
ARG VITE_API_MODE=remote
ARG VITE_ORGANIZATION_SLUG=qutcraft
ENV VITE_API_MODE=$VITE_API_MODE
ENV VITE_ORGANIZATION_SLUG=$VITE_ORGANIZATION_SLUG
RUN pnpm build

FROM nginx:1.27-alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
