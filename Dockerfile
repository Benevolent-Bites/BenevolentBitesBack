FROM golang:buster

# Set necessary environment variables
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
RUN echo ${API_KEY}
RUN echo $BRUH

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

# Move to /dist directory as the place for resulting binary folder
WORKDIR /dist

# Copy necessary files along with the binary
RUN mkdir cmd
RUN cp /build/cmd/cmd ./cmd
RUN cp -r /build/assets .
RUN cp -r /build/templates .

# Expose ports
EXPOSE 9000
EXPOSE 9001

# Install sudo
RUN apt-get update
RUN apt -y install sudo
RUN adduser --disabled-password --gecos '' docker
RUN adduser docker sudo
RUN echo '%sudo ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers
USER docker

# Start container
RUN sudo chmod +x -f ./cmd
CMD ["pwd"]


