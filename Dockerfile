FROM golang:buster

# Set necessary environment variables
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

# Move to working directory
WORKDIR /build

# Copy and download golang depedencies
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the code into the container
COPY . .

# Build server
RUN cd ./cmd && go build
WORKDIR /build/cmd

# Install sudo & Python
RUN apt-get update && apt -y install sudo python3-pip
RUN adduser --disabled-password --gecos '' docker
RUN adduser docker sudo
RUN echo '%sudo ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers
USER docker

# Make sure Python is installed
RUN pip3 install click requests

# Start container
RUN sudo chmod +x -f ./cmd
CMD ["sudo", "-E", "./cmd"]