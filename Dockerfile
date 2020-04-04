FROM golang:alpine

# Set necessary environment variables
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

# Move to working directory
WORKDIR /build

# Install git
RUN apk add --no-cache git

# Copy and download golang depedencies
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the code into the container
COPY . .

# Build server
RUN cd ./cmd && go build

# Move to /dist directory as the place for resulting binary folder
WORKDIR /dist

# Copy necessary files along with the binary
RUN mkdir cmd
RUN cp /build/cmd/cmd ./cmd
RUN cp -r /build/assets .
RUN cp -r /build/templates .

# Install Octopus CLI for deployment
RUN sudo apt update && sudo apt install --no-install-recommends gnupg curl ca-certificates apt-transport-https && \
    curl -sSfL https://apt.octopus.com/public.key | sudo apt-key add - && \
    sudo sh -c "echo deb https://apt.octopus.com/ stable main > /etc/apt/sources.list.d/octopus.com.list" && \
    sudo apt update && sudo apt install octopuscli

# Package /dist folder for release with Octopus
# See JenkinsFile for release process
RUN octo pack . --id="benevolent-back" --version="1.0.0" --format=zip
