# pull official base image
FROM node:14 as dev

# set working directory
WORKDIR /app
EXPOSE 3000

COPY package.json package-lock.json ./
RUN npm install

COPY cogment.yaml *.proto ./
RUN mkdir src
RUN npx cogment-js-sdk-generate cogment.yaml

# copy generated app
COPY . ./

# start app
CMD ["npm", "start"]