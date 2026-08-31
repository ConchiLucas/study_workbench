FROM node:20-alpine AS builder

WORKDIR /app

COPY rob_english_word_cloze_web/package.json rob_english_word_cloze_web/package-lock.json ./
RUN npm ci

COPY rob_english_word_cloze_web/. .
RUN npm run build

FROM nginx:1.27-alpine AS runner

COPY --from=builder /app/dist /usr/share/nginx/html
COPY deploy/frontend/rob_english_word_cloze_web/local_full/nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
