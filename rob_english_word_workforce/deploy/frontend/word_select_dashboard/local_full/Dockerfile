FROM node:20-alpine AS builder

WORKDIR /app

COPY word_select_dashboard/web-react/package.json word_select_dashboard/web-react/package-lock.json ./
RUN npm ci

COPY word_select_dashboard/web-react/. .
RUN npm run build

FROM nginx:1.27-alpine AS runner

COPY --from=builder /app/dist /usr/share/nginx/html
COPY deploy/frontend/word_select_dashboard/local_full/nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
