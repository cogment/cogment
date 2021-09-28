# pull official base image
FROM node:14 as dev

# set working directory
WORKDIR /app
EXPOSE 3000

# install modules
COPY package.json package-lock.json ./
COPY cogment.yaml ./
COPY *.proto ./

RUN mkdir src
RUN npm i --save-optional
RUN npx cogment-js-sdk-generate
RUN npm install

# copy generated app
COPY . ./

# generate code
RUN npx cogment-js-sdk-generate

# start app
CMD ["npm", "start"]

