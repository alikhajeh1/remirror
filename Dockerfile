# STEP 1
#
# Use Golang Image for building go app and using the artifacts in final image
FROM golang:1.11.4-alpine3.8 AS build

# * `GO111MODULE` This should not need changed.
# * `GOOS` This should not need changed.
# * `GOARCH` This should not need changed.
ENV GO111MODULE=on \
    GOOS=linux \
    GOARCH=amd64

# Install the base required packages for the image.
RUN apk add --update --no-cache \
      build-base \
      git \
      ca-certificates \
    && \
    mkdir -p /src

# First add modules list to better utilize caching
COPY go.mod /src/

# Change working directory (It's like cd)
WORKDIR /src

# Download go dependencies
RUN go mod download

# Copy source code to docker base image
COPY . /src

# Build components.
RUN go build -ldflags="-w -s" -o remirror

# STEP 2
#
#build a small image
FROM alpine:3.8

# TZ Timezone for the app.
ENV TZ=UTC

# Install the base required packages for the image.
RUN apk add --update --no-cache \
      tzdata \
      ca-certificates \
    && \
    cp --remove-destination /usr/share/zoneinfo/${TZ} /etc/localtime && \
    echo "${TZ}" > /etc/timezone

# Set the current directory for the Docker image to the Go bin path, since
# we will now run the Go app.
WORKDIR /app

# Expose any port your application listens on. You must use ports higher than 1024.
EXPOSE 8080

# Copy our static executable
COPY --from=build /src/remirror /app/
COPY remirror.hcl /app/

# Run the binary.
CMD ["/app/remirror"]
