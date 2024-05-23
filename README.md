# Tharsis API

Tharsis is a remote Terraform backend that provides state management and a full execution environment for running Terraform modules. It also includes additional capabilities to facilitate the management of Terraform workspaces within an organization.

- Configurable job executor plugin.
- Machine to Machine (M2M) authentication with service accounts.
- Managed identity support to securely authenticate with cloud providers (no credential storage).
- Users are not required to handle secrets.
- Compatible with the Terraform CLI remote backend.
- Ability to quickly cancel jobs.
- Support for uploading and downloading Terraform modules.
- Capable of being deployed as a Docker image.

## Get started

Instructions on building a binary from source can be found [here](https://gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-docs/-/blob/main/docs/setup/api/install.md).

## Installing locally (Requires Docker)

In order to use Tharsis locally, a `docker-compose.yml` file is available under the `docker-compose` directory. The use of this requires the installation of Docker. Learn more [here](https://docs.docker.com/get-docker/).

    1. Clone the repository and change to 'docker-compose' directory.
    2. Use 'docker compose up' command to launch Tharsis using Docker. We could additionally, pass in a `-d` flag to the command to launch everything in the background like so 'docker compose up -d'.

At this point, Docker should begin pulling down all the images needed to provision Tharsis. Once this is complete, the Tharsis UI will be available at `http://localhost:3000/`.

Now, the Tharsis CLI can be used to interact with Tharsis. To configure the CLI, use the following command to create a profile and sign in using the [KeyCloak](https://www.keycloak.org/) identity provider (IDP):

```bash
tharsis configure --profile local --endpoint-url http://localhost:6560
tharsis -p local sso login
```
At the KeyCloak login screen use `martian` for both the username and password to complete authentication.

**Congratulations! The Tharsis CLI is now ready to issue commands, use the `-h` flag for more info!**

## Running Tharsis API from local source

In order to run Tharsis API from local source we suggest you do the following:

    1. Stop the tharsis-api docker container if it is running.
    2. Copy the `env.example` file in the root folder and paste it as `.env`.
    3. Open the Tharsis API folder in Visual Studio Code.
    4. Install the recommended extensions.
    5. Click the `Run and Debug` menu on the left hand side of Visual Studio Code.
    6. Click the Start Debugging button next to Launch API.

At this point you can interact with the Tharsis UI at `http://localhost:3000/` and it will be communicating with your local Tharsis API.

## Documentation

- Tharsis API documentation is available at https://gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-docs/-/blob/main/docs.

## Security

If you've discovered a security vulnerability in the Tharsis API, please let us know by creating a **confidential** issue in this project.

## Statement of support

Please submit any bugs or feature requests for Tharsis. Of course, MR's are even better. :)

## License

Tharsis API is distributed under [Mozilla Public License v2.0](https://www.mozilla.org/en-US/MPL/2.0/).
