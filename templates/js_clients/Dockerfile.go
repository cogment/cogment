package js_clients

const DOCKERFILE = `
FROM node:12.4-alpine AS dev

WORKDIR /app

COPY clients/js/package.json /app/

RUN npm install

COPY clients/js /app

ENTRYPOINT ["npm"]
CMD ["start"]
`
