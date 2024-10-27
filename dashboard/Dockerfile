FROM node:20

WORKDIR /app

COPY package.json .
COPY package-lock.json .

RUN rm -rf node_modules
RUN npm i

COPY . .

## EXPOSE [Port you mentioned in the vite.config file]

LABEL org.opencontainers.image.source="https://github.com/b-harvest/metisian"

EXPOSE 5173/tcp

CMD ["npm", "run", "dev"]