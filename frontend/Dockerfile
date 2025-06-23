FROM node:20-slim AS builder

RUN apt-get update && apt-get install -y \
  git \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /app

ENV CYPRESS_INSTALL_BINARY=0

COPY package*.json ./
RUN npm install

COPY . .
RUN npm run build

FROM nginx:1.25-alpine AS production

RUN rm -rf /usr/share/nginx/html/*

COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
