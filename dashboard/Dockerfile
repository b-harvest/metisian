FROM node:23-alpine AS runner

WORKDIR /home/node/app

COPY --chown=node:node ./dist ./dist

RUN npm install -g serve

USER node

EXPOSE 3000
CMD ["serve", "-s", "dist", "-l", "3000"]
