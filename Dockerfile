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
RUN go get -u github.com/gin-contrib/cors
RUN go get -u github.com/gin-gonic/gin 
RUN go get -u github.com/google/uuid 
RUN go get -u github.com/sirupsen/logrus 
RUN go get -u go.mongodb.org/mongo-driver/mongo
RUN go get -u golang.org/x/crypto/bcrypt
RUN go get -u golang.org/x/oauth2/google 
RUN go get -u googlemaps.github.io/maps

# Copy the code into the container
COPY . .

# Build server
RUN cd ./cmd && go build

# Move to /dist directory as the place for resulting binary folder
WORKDIR /dist

# Copy binary from build to main folder
RUN cp /build/cmd .

# Export necessary ports (prod and dev)
EXPOSE 9000
EXPOSE 9001

# Command to run when starting the container
CMD ["/dist/main"]