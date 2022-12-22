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

## Documentation

- Tharsis API documentation is available at https://gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-docs/-/blob/main/docs.

## Security

If you've discovered a security vulnerability in the Tharsis API, please let us know by creating a **confidential** issue in this project.

## Statement of support

Please submit any bugs or feature requests for Tharsis.  Of course, MR's are even better.  :)

## License

Tharsis API is distributed under [Mozilla Public License v2.0](https://www.mozilla.org/en-US/MPL/2.0/).
