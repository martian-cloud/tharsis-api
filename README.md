# Tharsis API

Tharsis API is the core component of the Tharsis platform, an enterprise-scale Terraform backend that provides state management and a full execution environment for running Terraform modules. It offers a comprehensive solution for managing Terraform deployments, state, and workspaces within an organization. The API is designed as a stateless service, allowing for horizontal scaling and high availability deployments.

## Key Features

- **GraphQL API**:

  - Modern, flexible API with subscriptions for real-time updates
  - WebSocket support for live job logs and events
  - Comprehensive query, mutation, and subscription capabilities
  - GraphiQL interface for API exploration

- **Configurable Job Executor**:

  - Support for multiple execution environments:
    - Docker
    - Kubernetes with EKS configuration support
    - AWS ECS
    - Local execution
  - Runner agent architecture for distributed job execution
  - Job queuing and prioritization

- **Authentication & Authorization**:

  - Machine to Machine (M2M) authentication with service accounts
  - OIDC federation support
  - Role-based access control with fine-grained permissions
  - SCIM token support for identity management
  - Custom role creation with granular permission assignment

- **Managed Identities**:

  - Secure cloud provider authentication without credential storage
  - Support for AWS, Azure, and Tharsis federated identity types
  - Access rules to control identity assumption based on module attestations
  - Identity aliasing for cross-namespace access
  - Just-in-time credential generation

- **State Management**:

  - Secure Terraform state storage and versioning
  - Compatible with Terraform CLI remote backend
  - State locking to prevent concurrent modifications
  - State version history and rollback capabilities

- **Run Workflow**:

  - Plan, apply, and destroy operations
  - Workspace assessment capabilities
  - Quick job cancellation
  - Real-time job logs via WebSocket subscriptions
  - Configurable run triggers and notifications

- **Module & Provider Registries**:

  - Built-in Terraform module registry
  - Module attestation support using in-toto specification
  - Built-in Terraform provider registry
  - Provider platform mirroring
  - Semantic versioning support
  - Module dependency management

- **VCS Integration**:

  - GitHub and GitLab support
  - Automatic run triggering on repository changes
  - OAuth token management
  - Branch and path filtering
  - Pull/Merge request integration

- **Federated Registry Support**:

  - Registry federation for cross-organization module/provider sharing
  - Token-based authentication between federated registries
  - Centralized module/provider distribution

- **Resource Management**:

  - Hierarchical group and workspace organization
  - Namespace variable management
  - Resource limits and quotas
  - Activity event tracking and auditing

- **Operational Features**:
  - Maintenance mode for controlled system updates
  - Resource quotas and limits
  - Comprehensive activity logging
  - Team management and access control
  - Pluggable architecture for custom job executors

## Architecture

Tharsis API is built with a layered architecture that separates concerns and promotes maintainability:

- **API Layer**: GraphQL API with comprehensive query, mutation, and subscription support
- **Service Layer**: Core business logic and service implementations
- **Data Access Layer**: Database abstraction for persistent storage in PostgreSQL
- **Job Execution Layer**: Interfaces with various execution environments (Docker, K8s, ECS)
- **Authentication Layer**: Handles various authentication mechanisms including OIDC and service accounts

As a stateless service, the API can be horizontally scaled to handle increased load, with state maintained in the database and external storage systems.

## API Capabilities

The Tharsis API provides a comprehensive set of GraphQL operations:

- **Queries**: Retrieve information about workspaces, groups, runs, jobs, modules, providers, and more
- **Mutations**: Create, update, and delete resources, trigger runs, manage permissions, and more
- **Subscriptions**: Real-time updates for job logs, run events, workspace events, and more

## Get started

For comprehensive documentation, visit the [Tharsis Documentation Site](https://tharsis.martian-cloud.io/).

Instructions on building a binary from source can be found [here](https://tharsis.martian-cloud.io/setup).

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

- Official Tharsis documentation is available at [https://tharsis.martian-cloud.io/](https://tharsis.martian-cloud.io/)

## Security

If you've discovered a security vulnerability in the Tharsis API, please let us know by creating a **confidential** issue in this project.

## Statement of support

Please submit any bugs or feature requests for Tharsis. Of course, MR's are even better. :)

## License

Tharsis API is distributed under [Mozilla Public License v2.0](https://www.mozilla.org/en-US/MPL/2.0/).
