FROM node:20-alpine AS builder

WORKDIR /app

COPY rob_english_word_front/package.json rob_english_word_front/package-lock.json ./
RUN npm ci

COPY rob_english_word_front/. .
RUN npm run build

FROM nginx:1.27-alpine AS runner

COPY --from=builder /app/dist /usr/share/nginx/html
COPY deploy/frontend/rob_english_word_front/local_full/nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
