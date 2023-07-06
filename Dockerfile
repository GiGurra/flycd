FROM ubuntu:22.04

RUN apt update
RUN apt dist-upgrade -y
RUN apt install build-essential -y
RUN apt install wget -y

# Install Go
RUN wget https://go.dev/dl/go1.20.5.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.20.5.linux-amd64.tar.gz
ENV PATH="/usr/local/go/bin:${PATH}"

# Install yaml tools
RUN go install github.com/sclevine/yj/v5@v5.1.0

# Copy over the app
COPY . /flycd
WORKDIR /flycd

# Build the app
RUN go build -o flycd

# Run the app
CMD ["./flycd"]

