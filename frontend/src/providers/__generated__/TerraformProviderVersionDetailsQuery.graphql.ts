/**
 * @generated SignedSource<<93ba5e2a2ffd704a7b3eb11afa618e3a>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type TerraformProviderVersionDetailsQuery$variables = {
  providerName: string;
  registryNamespace: string;
  version?: string | null | undefined;
};
export type TerraformProviderVersionDetailsQuery$data = {
  readonly terraformProviderVersion: {
    readonly id: string;
    readonly " $fragmentSpreads": FragmentRefs<"TerraformProviderVersionDetailsIndexFragment_details">;
  } | null | undefined;
};
export type TerraformProviderVersionDetailsQuery = {
  response: TerraformProviderVersionDetailsQuery$data;
  variables: TerraformProviderVersionDetailsQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "providerName"
},
v1 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "registryNamespace"
},
v2 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "version"
},
v3 = [
  {
    "kind": "Variable",
    "name": "providerName",
    "variableName": "providerName"
  },
  {
    "kind": "Variable",
    "name": "registryNamespace",
    "variableName": "registryNamespace"
  },
  {
    "kind": "Variable",
    "name": "version",
    "variableName": "version"
  }
],
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/),
      (v2/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "TerraformProviderVersionDetailsQuery",
    "selections": [
      {
        "alias": null,
        "args": (v3/*: any*/),
        "concreteType": "TerraformProviderVersion",
        "kind": "LinkedField",
        "name": "terraformProviderVersion",
        "plural": false,
        "selections": [
          (v4/*: any*/),
          {
            "args": null,
            "kind": "FragmentSpread",
            "name": "TerraformProviderVersionDetailsIndexFragment_details"
          }
        ],
        "storageKey": null
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [
      (v1/*: any*/),
      (v0/*: any*/),
      (v2/*: any*/)
    ],
    "kind": "Operation",
    "name": "TerraformProviderVersionDetailsQuery",
    "selections": [
      {
        "alias": null,
        "args": (v3/*: any*/),
        "concreteType": "TerraformProviderVersion",
        "kind": "LinkedField",
        "name": "terraformProviderVersion",
        "plural": false,
        "selections": [
          (v4/*: any*/),
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "version",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "readme",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "shaSumsUploaded",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "shaSumsSigUploaded",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "concreteType": "ResourceMetadata",
            "kind": "LinkedField",
            "name": "metadata",
            "plural": false,
            "selections": [
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "trn",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "createdAt",
                "storageKey": null
              }
            ],
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "concreteType": "TerraformProvider",
            "kind": "LinkedField",
            "name": "provider",
            "plural": false,
            "selections": [
              (v4/*: any*/),
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
                "name": "registryNamespace",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "private",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "repositoryUrl",
                "storageKey": null
              }
            ],
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "createdBy",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "gpgKeyId",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "protocols",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "latest",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "concreteType": "TerraformProviderPlatform",
            "kind": "LinkedField",
            "name": "platforms",
            "plural": true,
            "selections": [
              (v4/*: any*/),
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "os",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "arch",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "binaryUploaded",
                "storageKey": null
              }
            ],
            "storageKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "5d0b9c031e8f7be0aa033353c724d751",
    "id": null,
    "metadata": {},
    "name": "TerraformProviderVersionDetailsQuery",
    "operationKind": "query",
    "text": "query TerraformProviderVersionDetailsQuery(\n  $registryNamespace: String!\n  $providerName: String!\n  $version: String\n) {\n  terraformProviderVersion(registryNamespace: $registryNamespace, providerName: $providerName, version: $version) {\n    id\n    ...TerraformProviderVersionDetailsIndexFragment_details\n  }\n}\n\nfragment TerraformProviderVersionDetailsIndexFragment_details on TerraformProviderVersion {\n  id\n  version\n  readme\n  shaSumsUploaded\n  shaSumsSigUploaded\n  metadata {\n    trn\n  }\n  provider {\n    id\n    name\n    registryNamespace\n    private\n    ...TerraformProviderVersionListFragment_provider\n  }\n  ...TerraformProviderVersionDetailsSidebarFragment_details\n}\n\nfragment TerraformProviderVersionDetailsSidebarFragment_details on TerraformProviderVersion {\n  version\n  createdBy\n  gpgKeyId\n  protocols\n  latest\n  platforms {\n    id\n    os\n    arch\n    binaryUploaded\n  }\n  metadata {\n    createdAt\n  }\n  provider {\n    id\n    name\n    registryNamespace\n    private\n    repositoryUrl\n  }\n}\n\nfragment TerraformProviderVersionListFragment_provider on TerraformProvider {\n  id\n}\n"
  }
};
})();

(node as any).hash = "07b31df8fb56fb31346851645bf111f0";

export default node;
