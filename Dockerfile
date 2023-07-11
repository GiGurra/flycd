FROM ubuntu:22.04

# Install dependencies
RUN apt update
RUN apt dist-upgrade -y
RUN apt install build-essential -y
RUN apt install wget -y
RUN apt install git -y
RUN apt install jq -y
RUN apt install curl -y

# Install Go
RUN wget https://go.dev/dl/go1.20.5.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.20.5.linux-amd64.tar.gz
ENV PATH="/usr/local/go/bin:${PATH}"
ENV PATH="/root/go/bin:${PATH}"

# Install yaml tools
RUN go install github.com/sclevine/yj/v5@v5.1.0

# Install flyctl/fly.io cli
RUN curl -L https://fly.io/install.sh | sh

ENV FLYCTL_INSTALL="/root/.fly"
ENV PATH="$FLYCTL_INSTALL/bin:$PATH"

# Copy over the app, excl the projects folder
RUN mkdir -p /flycd
COPY ./cmd /flycd/cmd
COPY ./internal /flycd/internal
COPY ./main.go /flycd/main.go
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

# grab the latest version of flyctl
RUN flyctl version upgrade

# store known hosts for github.com and bitbucket.org
RUN mkdir -p /root/.ssh
RUN ssh-keyscan github.com >> /root/.ssh/known_hosts
RUN ssh-keyscan bitbucket.org >> /root/.ssh/known_hosts

# Lastly, copy the latest version of the projects folder
COPY ./projects /flycd/projects

# Run the app
ENTRYPOINT ["flycd"]
CMD ["monitor", "projects"]

