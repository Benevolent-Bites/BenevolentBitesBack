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

# Install Octopus CLI for deployment
RUN [ -z "$DOTNET_BUNDLE_EXTRACT_BASE_DIR" ] && export DOTNET_BUNDLE_EXTRACT_BASE_DIR="${XDG_CACHE_HOME:-"$HOME"/.cache}/dotnet_bundle_extract"
RUN apt update && apt install -y --no-install-recommends gnupg curl ca-certificates apt-transport-https && \
    curl -sSfL https://apt.octopus.com/public.key | apt-key add - && \
    sh -c "echo deb https://apt.octopus.com/ stable main > /etc/apt/sources.list.d/octopus.com.list" && \
    apt update && apt install -y octopuscli

# Package /dist folder for release with Octopus
# See JenkinsFile for release process
RUN octo pack . --id="benevolent-back" --version="1.0.0" --format=zip

# Install sudo for Jenkins
RUN apt -y install sudo
RUN echo "jenkins ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers
RUN chmod 777 -R /var/tmp/.net
RUN chmod 777 -R /dist


