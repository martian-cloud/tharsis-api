/**
 * @generated SignedSource<<d8c7330a6713468daf53a2b4705c8cc9>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type EditTerraformModuleQuery$variables = {
  moduleName: string;
  registryNamespace: string;
  system: string;
};
export type EditTerraformModuleQuery$data = {
  readonly terraformModule: {
    readonly id: string;
    readonly labels: ReadonlyArray<{
      readonly key: string;
      readonly value: string;
    }>;
    readonly name: string;
    readonly private: boolean;
    readonly system: string;
  } | null | undefined;
};
export type EditTerraformModuleQuery = {
  response: EditTerraformModuleQuery$data;
  variables: EditTerraformModuleQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "moduleName"
},
v1 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "registryNamespace"
},
v2 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "system"
},
v3 = [
  {
    "alias": null,
    "args": [
      {
        "kind": "Variable",
        "name": "moduleName",
        "variableName": "moduleName"
      },
      {
        "kind": "Variable",
        "name": "registryNamespace",
        "variableName": "registryNamespace"
      },
      {
        "kind": "Variable",
        "name": "system",
        "variableName": "system"
      }
    ],
    "concreteType": "TerraformModule",
    "kind": "LinkedField",
    "name": "terraformModule",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "id",
        "storageKey": null
      },
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
        "name": "system",
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
        "concreteType": "TerraformModuleLabel",
        "kind": "LinkedField",
        "name": "labels",
        "plural": true,
        "selections": [
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
        "storageKey": null
      }
    ],
    "storageKey": null
  }
];
return {
  "fragment": {
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/),
      (v2/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "EditTerraformModuleQuery",
    "selections": (v3/*: any*/),
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
    "name": "EditTerraformModuleQuery",
    "selections": (v3/*: any*/)
  },
  "params": {
    "cacheID": "6510af5e65b40043af07bb1ebb732df4",
    "id": null,
    "metadata": {},
    "name": "EditTerraformModuleQuery",
    "operationKind": "query",
    "text": "query EditTerraformModuleQuery(\n  $registryNamespace: String!\n  $moduleName: String!\n  $system: String!\n) {\n  terraformModule(registryNamespace: $registryNamespace, moduleName: $moduleName, system: $system) {\n    id\n    name\n    system\n    private\n    labels {\n      key\n      value\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "24d253c1a7ef81e2f5f7fcef374a5d8a";

export default node;
