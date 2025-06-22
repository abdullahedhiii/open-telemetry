FROM node:24-alpine AS builder

WORKDIR /my-frontend-app

COPY package*.json ./

RUN npm install

ARG VITE_API_URL
ENV VITE_API_URL=$VITE_API_URL

COPY . .

RUN npm run build

FROM nginx:alpine

COPY --from=builder /my-frontend-app/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx","-g","daemon off;"]