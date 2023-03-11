# Development container

Development container tailored for Gossamer that can be used with VSCode on Linux, Windows and OSX.

It is based on [`qmcgaw/godevcontainer`](https://github.com/qdm12/godevcontainer) and notably has:

- `CGO_ENABLED=1` set to work with the wasmer C bindings
- `procps` and `cpulimit` installed to limit the CPU usage of a Gossamer node
- NodeJS 14 installed to run polkadotjs end to end tests
- [github.com/google/addlicense](https://github.com/google/addlicense) installed to add and maintain copyright notices to source files
- Protoc and [protoc-gen-go](google.golang.org/protobuf/cmd/protoc-gen-go) to generate Go source files from proto files

## Requirements

- [VS code](https://code.visualstudio.com/download) installed
- [VS code remote containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers) installed
- [Docker](https://www.docker.com/products/docker-desktop) installed and running
- [Docker Compose](https://docs.docker.com/compose/install/) installed

## Setup

1. Create the following files on your host if you don't have them:

    ```sh
    touch ~/.gitconfig ~/.zsh_history
    ```

    Note that the development container will create the empty directories `~/.docker`, `~/.ssh` and `~/.kube` if you don't have them.

1. **For Docker on OSX or Windows without WSL**: ensure your home directory `~` is accessible by Docker.
1. **For Docker on Windows without WSL:** if you want to use SSH keys, bind mount your host `~/.ssh` to `/tmp/.ssh` instead of `~/.ssh` by changing the `volumes` section in the [docker-compose.yml](docker-compose.yml).
1. Open the command palette in Visual Studio Code (CTRL+SHIFT+P).
1. Select `Remote-Containers: Open Folder in Container...` and choose the project directory.

## Customization

### Customize the image

You can make changes to the [Dockerfile](Dockerfile) and then rebuild the image.

To rebuild the image, open the VSCode command palette (`CTRL`+`SHIFT`+`P`), select `Remote-Containers: Rebuild and reopen in container`

### Customize VS code settings

You can customize **settings** and **extensions** in the [devcontainer.json](devcontainer.json) definition file. You will have to re-build the container for them to take effect. Alternatively you can still use the `.vscode` directory in the repository for user settings that take precedence.

### Entrypoint script

You can bind mount a shell script to `/home/vscode/.welcome.sh` to replace the current welcome script.

### Publish a port

To access a port from your host to your development container, publish a port in [docker-compose.yml](docker-compose.yml). You can also now do it directly with VSCode without restarting the container.

### Run other services

1. Modify [docker-compose.yml](docker-compose.yml) to launch other services at the same time as this development container, such as a test database:

    ```yml
      database:
        image: postgres
        restart: always
        environment:
          POSTGRES_PASSWORD: password
    ```

1. In [devcontainer.json](devcontainer.json), change the line `"runServices": ["vscode"],` to `"runServices": ["vscode", "database"],`.
1. In the VS code command palette, rebuild the container.
