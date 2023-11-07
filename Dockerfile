# Build the agent binary.
FROM golang:1.19.4 as builder

WORKDIR /

COPY ./LICENSE /licenses
COPY bin/cloudops /cloudops

EXPOSE 8090

ENTRYPOINT ["./cloudops"]