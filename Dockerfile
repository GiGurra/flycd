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

# Copy over the app
COPY . /flycd
WORKDIR /flycd
ENV PATH="/flycd:${PATH}"

# Build the app
RUN go build -o flycd

# We actually always want to do this last, so we always get a new version of flyctl
RUN curl -L https://fly.io/install.sh | sh

ENV FLYCTL_INSTALL="/root/.fly"
ENV PATH="$FLYCTL_INSTALL/bin:$PATH"

# Run the app
ENTRYPOINT ["flycd"]
CMD ["monitor"]

