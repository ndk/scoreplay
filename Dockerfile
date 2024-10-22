FROM golang:1.23 AS base
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .

FROM base AS build
WORKDIR /app
RUN go build -o bin/scoreplay .

FROM base

ARG user=appuser
ARG group=appuser
ARG uid=1000
ARG gid=1000
RUN groupadd -g ${gid} ${group} && useradd -u ${uid} -g ${group} -s /bin/sh ${user}
RUN chown ${user}:${user} .
USER ${user}

WORKDIR /app
COPY --from=build /app/bin/scoreplay .

EXPOSE 8080
HEALTHCHECK --interval=1s --timeout=1s --start-period=1ms --retries=10 \
  CMD curl --fail ${APIURL}/health || exit 1
ENTRYPOINT /app/scoreplay
