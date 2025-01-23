# create_snapshot

A utility script to create a snapshot of a Gossamer node's database once a specified block height is reached.

## Requirements

- The Gossamer node must be running inside a Docker container.
- The container's name must contain the substring `gossamer`.
- The Gossamer node must expose Prometheus metrics on port 9876.
- The script requires access to the Docker socket (/var/run/docker.sock) to manage the Gossamer container.

## Usage

The script is included in the Docker image. You can run it inside Docker by making the Docker socket available to the
container. Note that this is a potential security risk as the Docker socket can be used to execute arbitrary commands on
the host system. So make sure to audit the script before running it. :)
Assuming the Gossamer container is using the "host" network mode, the data directory is mounted at `/data/gossamer_data`
and the image is called `gossamer:latest`, you can run the script as follows:

```bash
docker run --network host --rm --name snapshotter \
  -v /data/gossamer_data:/gossamer_data \
  -v <snapshot destination>:/snapshots:rw \
  -v /var/run/docker.sock:/var/run/docker.sock \
  --entrypoint sh \
  gossamer:latest \
  -c '/gossamer/bin/create_snapshot <approximate block height> /gossamer_data/db /snapshots'
```

Not that the second parameter is the path to the database directory `db`, inside the container. Replace
`<approximate block height>` and `<snapshot destination>` with the appropriate values. If you don't care about the exact
block height and want to take a snapshot immediately, you can pass `1` as the first parameter.

If the host uses the same OS and CPU architecture as the Gossamer image, you can also copy the binary to the host and
run it directly:

```bash
$ CONTAINER_ID=$(docker create gossamer:latest)
$ docker cp ${CONTAINER_ID}:/gossamer/bin/create_snapshot ./create_snapshot
Successfully copied 10.2MB to /apps/gossamer/create_snapshot
$ docker rm $CONTAINER_ID
8c3f09bcab0b013c873382259094f65efc3fb679a0b129e01f9849c603a0ca85
$ ./create_snapshot
usage: create_snapshot <approximate block height> <db location> <snapshot destination>
```
