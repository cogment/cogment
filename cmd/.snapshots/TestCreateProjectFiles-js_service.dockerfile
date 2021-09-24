# pull official base image
FROM node:14 as dev

# set working directory
WORKDIR /app
EXPOSE 3000

# install modules
COPY package.json package-lock.json ./
RUN npm install --save-optional

# copy generated app
COPY . ./

# generate code
RUN npx cogment-js-sdk-generate

# start app
CMD ["npm", "start"]

