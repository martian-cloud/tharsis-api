/**
 * @generated SignedSource<<74d5296459818b256c4735008c050ebe>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type AdminAreaConfigurationSettingsQuery$variables = Record<PropertyKey, never>;
export type AdminAreaConfigurationSettingsQuery$data = {
  readonly config: {
    readonly adminLogTailStorePluginData: ReadonlyArray<{
      readonly key: string;
      readonly value: string;
    }>;
    readonly adminLogTailStorePluginType: string;
    readonly adminUserEmail: string;
    readonly aiEnabled: boolean;
    readonly asymmetricSigningKeyDecommissionPeriodDays: number;
    readonly asymmetricSigningKeyRotationPeriodDays: number;
    readonly asyncTaskTimeout: number;
    readonly cliLoginOIDCClientID: string | null | undefined;
    readonly cliLoginOIDCScopes: string | null | undefined;
    readonly corsAllowedOrigins: string;
    readonly dbAutoMigrateEnabled: boolean;
    readonly dbHost: string;
    readonly dbMaxConnections: number;
    readonly dbName: string;
    readonly dbPort: number;
    readonly dbSslMode: string;
    readonly disableSensitiveVariableFeature: boolean;
    readonly emailClientPluginData: ReadonlyArray<{
      readonly key: string;
      readonly value: string;
    }>;
    readonly emailClientPluginType: string;
    readonly emailFooter: string | null | undefined;
    readonly federatedRegistryTrustPolicies: ReadonlyArray<{
      readonly audience: string | null | undefined;
      readonly groupGlobPatterns: ReadonlyArray<string>;
      readonly issuerUrl: string;
      readonly subject: string | null | undefined;
    }>;
    readonly httpRateLimit: number;
    readonly internalRunners: ReadonlyArray<{
      readonly jobDispatcherData: ReadonlyArray<{
        readonly key: string;
        readonly value: string;
      }>;
      readonly jobDispatcherType: string;
      readonly name: string;
    }>;
    readonly jwsProviderPluginData: ReadonlyArray<{
      readonly key: string;
      readonly value: string;
    }>;
    readonly jwsProviderPluginType: string;
    readonly jwtIssuerUrl: string;
    readonly maxGraphQlComplexity: number;
    readonly mcpServerConfig: {
      readonly enabledTools: ReadonlyArray<string>;
      readonly enabledToolsets: ReadonlyArray<string>;
      readonly readOnly: boolean;
    };
    readonly moduleRegistryMaxUploadSize: number;
    readonly oauthProviders: ReadonlyArray<{
      readonly clientId: string;
      readonly issuerUrl: string;
      readonly scope: string;
      readonly usernameClaim: string;
    }>;
    readonly objectStorePluginData: ReadonlyArray<{
      readonly key: string;
      readonly value: string;
    }>;
    readonly objectStorePluginType: string;
    readonly oidcInternalIdentityProviderClientID: string;
    readonly otelTraceCollectorHost: string | null | undefined;
    readonly otelTraceCollectorPort: number;
    readonly otelTraceEnabled: boolean;
    readonly otelTraceType: string | null | undefined;
    readonly rateLimitStorePluginData: ReadonlyArray<{
      readonly key: string;
      readonly value: string;
    }>;
    readonly rateLimitStorePluginType: string;
    readonly secretManagerPluginData: ReadonlyArray<{
      readonly key: string;
      readonly value: string;
    }>;
    readonly secretManagerPluginType: string;
    readonly sensitiveFields: ReadonlyArray<string>;
    readonly serverPort: string;
    readonly serviceAccountClientSecretMaxExpirationDays: number;
    readonly serviceDiscoveryHost: string;
    readonly terraformCliVersionConstraint: string;
    readonly tharsisApiUrl: string;
    readonly tharsisSupportUrl: string;
    readonly tharsisUiUrl: string;
    readonly tlsCertFile: string;
    readonly tlsEnabled: boolean;
    readonly tlsKeyFile: string;
    readonly userSessionAccessTokenExpirationMinutes: number;
    readonly userSessionMaxSessionsPerUser: number;
    readonly userSessionRefreshTokenExpirationMinutes: number;
    readonly vcsRepositorySizeLimit: number;
    readonly workspaceAssessmentIntervalHours: number;
    readonly workspaceAssessmentRunLimit: number;
  };
};
export type AdminAreaConfigurationSettingsQuery = {
  response: AdminAreaConfigurationSettingsQuery$data;
  variables: AdminAreaConfigurationSettingsQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "issuerUrl",
  "storageKey": null
},
v1 = [
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "key",
    "storageKey": null
  },
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "value",
    "storageKey": null
  }
],
v2 = [
  {
    "alias": null,
    "args": null,
    "concreteType": "Config",
    "kind": "LinkedField",
    "name": "config",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "serverPort",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "tharsisApiUrl",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "tharsisUiUrl",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "tharsisSupportUrl",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "serviceDiscoveryHost",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "corsAllowedOrigins",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "tlsEnabled",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "httpRateLimit",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "jwtIssuerUrl",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "oidcInternalIdentityProviderClientID",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "cliLoginOIDCClientID",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "cliLoginOIDCScopes",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "concreteType": "IdpConfig",
        "kind": "LinkedField",
        "name": "oauthProviders",
        "plural": true,
        "selections": [
          (v0/*: any*/),
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "clientId",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "usernameClaim",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "scope",
            "storageKey": null
          }
        ],
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "userSessionAccessTokenExpirationMinutes",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "userSessionRefreshTokenExpirationMinutes",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "userSessionMaxSessionsPerUser",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "maxGraphQlComplexity",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "moduleRegistryMaxUploadSize",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "asyncTaskTimeout",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "vcsRepositorySizeLimit",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "serviceAccountClientSecretMaxExpirationDays",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "terraformCliVersionConstraint",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "workspaceAssessmentIntervalHours",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "workspaceAssessmentRunLimit",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "asymmetricSigningKeyRotationPeriodDays",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "asymmetricSigningKeyDecommissionPeriodDays",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "aiEnabled",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "disableSensitiveVariableFeature",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "emailFooter",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "objectStorePluginType",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "rateLimitStorePluginType",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "jwsProviderPluginType",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "secretManagerPluginType",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "emailClientPluginType",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "adminLogTailStorePluginType",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "concreteType": "KeyValueEntry",
        "kind": "LinkedField",
        "name": "objectStorePluginData",
        "plural": true,
        "selections": (v1/*: any*/),
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "concreteType": "KeyValueEntry",
        "kind": "LinkedField",
        "name": "rateLimitStorePluginData",
        "plural": true,
        "selections": (v1/*: any*/),
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "concreteType": "KeyValueEntry",
        "kind": "LinkedField",
        "name": "jwsProviderPluginData",
        "plural": true,
        "selections": (v1/*: any*/),
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "concreteType": "KeyValueEntry",
        "kind": "LinkedField",
        "name": "secretManagerPluginData",
        "plural": true,
        "selections": (v1/*: any*/),
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "concreteType": "KeyValueEntry",
        "kind": "LinkedField",
        "name": "emailClientPluginData",
        "plural": true,
        "selections": (v1/*: any*/),
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "concreteType": "KeyValueEntry",
        "kind": "LinkedField",
        "name": "adminLogTailStorePluginData",
        "plural": true,
        "selections": (v1/*: any*/),
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "dbHost",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "dbName",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "dbSslMode",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "dbPort",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "dbMaxConnections",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "dbAutoMigrateEnabled",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "tlsCertFile",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "tlsKeyFile",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "adminUserEmail",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "otelTraceEnabled",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "otelTraceType",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "otelTraceCollectorHost",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "otelTraceCollectorPort",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "concreteType": "FederatedRegistryTrustPolicy",
        "kind": "LinkedField",
        "name": "federatedRegistryTrustPolicies",
        "plural": true,
        "selections": [
          (v0/*: any*/),
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "subject",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "audience",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "groupGlobPatterns",
            "storageKey": null
          }
        ],
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "concreteType": "RunnerConfig",
        "kind": "LinkedField",
        "name": "internalRunners",
        "plural": true,
        "selections": [
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "name",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "jobDispatcherType",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "concreteType": "KeyValueEntry",
            "kind": "LinkedField",
            "name": "jobDispatcherData",
            "plural": true,
            "selections": (v1/*: any*/),
            "storageKey": null
          }
        ],
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "concreteType": "MCPServerConfig",
        "kind": "LinkedField",
        "name": "mcpServerConfig",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "enabledToolsets",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "enabledTools",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "readOnly",
            "storageKey": null
          }
        ],
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "sensitiveFields",
        "storageKey": null
      }
    ],
    "storageKey": null
  }
];
return {
  "fragment": {
    "argumentDefinitions": [],
    "kind": "Fragment",
    "metadata": null,
    "name": "AdminAreaConfigurationSettingsQuery",
    "selections": (v2/*: any*/),
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [],
    "kind": "Operation",
    "name": "AdminAreaConfigurationSettingsQuery",
    "selections": (v2/*: any*/)
  },
  "params": {
    "cacheID": "bcef3153aa7829882e5f29c8612546f0",
    "id": null,
    "metadata": {},
    "name": "AdminAreaConfigurationSettingsQuery",
    "operationKind": "query",
    "text": "query AdminAreaConfigurationSettingsQuery {\n  config {\n    serverPort\n    tharsisApiUrl\n    tharsisUiUrl\n    tharsisSupportUrl\n    serviceDiscoveryHost\n    corsAllowedOrigins\n    tlsEnabled\n    httpRateLimit\n    jwtIssuerUrl\n    oidcInternalIdentityProviderClientID\n    cliLoginOIDCClientID\n    cliLoginOIDCScopes\n    oauthProviders {\n      issuerUrl\n      clientId\n      usernameClaim\n      scope\n    }\n    userSessionAccessTokenExpirationMinutes\n    userSessionRefreshTokenExpirationMinutes\n    userSessionMaxSessionsPerUser\n    maxGraphQlComplexity\n    moduleRegistryMaxUploadSize\n    asyncTaskTimeout\n    vcsRepositorySizeLimit\n    serviceAccountClientSecretMaxExpirationDays\n    terraformCliVersionConstraint\n    workspaceAssessmentIntervalHours\n    workspaceAssessmentRunLimit\n    asymmetricSigningKeyRotationPeriodDays\n    asymmetricSigningKeyDecommissionPeriodDays\n    aiEnabled\n    disableSensitiveVariableFeature\n    emailFooter\n    objectStorePluginType\n    rateLimitStorePluginType\n    jwsProviderPluginType\n    secretManagerPluginType\n    emailClientPluginType\n    adminLogTailStorePluginType\n    objectStorePluginData {\n      key\n      value\n    }\n    rateLimitStorePluginData {\n      key\n      value\n    }\n    jwsProviderPluginData {\n      key\n      value\n    }\n    secretManagerPluginData {\n      key\n      value\n    }\n    emailClientPluginData {\n      key\n      value\n    }\n    adminLogTailStorePluginData {\n      key\n      value\n    }\n    dbHost\n    dbName\n    dbSslMode\n    dbPort\n    dbMaxConnections\n    dbAutoMigrateEnabled\n    tlsCertFile\n    tlsKeyFile\n    adminUserEmail\n    otelTraceEnabled\n    otelTraceType\n    otelTraceCollectorHost\n    otelTraceCollectorPort\n    federatedRegistryTrustPolicies {\n      issuerUrl\n      subject\n      audience\n      groupGlobPatterns\n    }\n    internalRunners {\n      name\n      jobDispatcherType\n      jobDispatcherData {\n        key\n        value\n      }\n    }\n    mcpServerConfig {\n      enabledToolsets\n      enabledTools\n      readOnly\n    }\n    sensitiveFields\n  }\n}\n"
  }
};
})();

(node as any).hash = "f22a62d94df686bacd956647a0aa95b1";

export default node;
