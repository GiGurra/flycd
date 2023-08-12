FROM golang:1.21.0-alpine3.18 as builder

# Copy over the app, excl the projects folder
RUN mkdir -p /flycd
COPY ./cmd /flycd/cmd
COPY ./pkg /flycd/pkg
COPY ./*.go /flycd/
COPY ./go.mod /flycd/go.mod
COPY ./go.sum /flycd/go.sum
COPY ./LICENSE /flycd/LICENSE
COPY ./Dockerfile /flycd/Dockerfile
COPY ./README.md /flycd/README.md
WORKDIR /flycd
ENV PATH="/flycd:${PATH}"

# Download the dependencies. Done in a separate step so we can cache it.
RUN go mod download

# Build the latest version of the app
RUN go build -o flycd


FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache bash curl file openssh-client git

# Copy over the built app from the builder stage
COPY --from=builder /flycd/flycd /flycd/flycd
WORKDIR /flycd
ENV PATH="/flycd:${PATH}"

# Install flycd cli last to ensure it's the latest version
RUN curl -L https://fly.io/install.sh | sh
ENV FLYCTL_INSTALL="/root/.fly"
ENV PATH="$FLYCTL_INSTALL/bin:$PATH"
RUN fly version upgrade

# same thing for storing known hosts for github.com and bitbucket.org, do it last to ensure it's the latest version
RUN mkdir -p /root/.ssh
RUN ssh-keyscan github.com >> /root/.ssh/known_hosts
RUN ssh-keyscan bitbucket.org >> /root/.ssh/known_hosts

# Run the app
EXPOSE 80
ENTRYPOINT ["flycd"]
CMD ["monitor", "projects"]
