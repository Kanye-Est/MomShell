FROM node:24-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --ignore-scripts
COPY index.html tsconfig.json tsconfig.node.json vite.config.ts env.d.ts eslint.config.js ./
COPY src/ src/
COPY public/ public/
ARG VITE_API_BASE_URL=
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
RUN npm run build

FROM nginx:alpine
ENV MOMSHELL_HTTP_PORT=80 MOMSHELL_BACKEND=host.docker.internal:8000 NGINX_ENVSUBST_FILTER=^MOMSHELL_
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/templates/default.conf.template
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
