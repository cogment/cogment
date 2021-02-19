# pull official base image
FROM node:14 as dev

ARG PROTOC_URL=https://github.com/protocolbuffers/protobuf/releases/download/v3.14.0/protoc-3.14.0-linux-x86_64.zip
ENV PROTOC_URL=${PROTOC_URL}

RUN curl -LSso /tmp/protoc.zip ${PROTOC_URL} \
  && unzip -d /usr/local/ /tmp/protoc.zip \
  && rm -rf /tmp/protoc.zip

# set working directory
WORKDIR /app
EXPOSE 3000

# install modules
COPY package.json package-lock.json ./
RUN npm install

# copy generated app
COPY . ./

# start app
CMD ["npm", "start"]
