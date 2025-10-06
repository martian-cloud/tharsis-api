/**
 * @generated SignedSource<<3be6f781b67244a0210af42fbabe0b8f>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ModuleVersionAutocompleteQuery$variables = {
  moduleName: string;
  registryNamespace: string;
  system: string;
  versionSearch?: string | null | undefined;
};
export type ModuleVersionAutocompleteQuery$data = {
  readonly terraformModule: {
    readonly versions: {
      readonly edges: ReadonlyArray<{
        readonly node: {
          readonly version: string;
        } | null | undefined;
      } | null | undefined> | null | undefined;
    };
  } | null | undefined;
};
export type ModuleVersionAutocompleteQuery = {
  response: ModuleVersionAutocompleteQuery$data;
  variables: ModuleVersionAutocompleteQuery$variables;
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
v3 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "versionSearch"
},
v4 = [
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
v5 = [
  {
    "kind": "Literal",
    "name": "first",
    "value": 50
  },
  {
    "kind": "Variable",
    "name": "search",
    "variableName": "versionSearch"
  },
  {
    "kind": "Literal",
    "name": "sort",
    "value": "CREATED_AT_DESC"
  }
],
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "version",
  "storageKey": null
},
v7 = {
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
      (v2/*: any*/),
      (v3/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "ModuleVersionAutocompleteQuery",
    "selections": [
      {
        "alias": null,
        "args": (v4/*: any*/),
        "concreteType": "TerraformModule",
        "kind": "LinkedField",
        "name": "terraformModule",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": (v5/*: any*/),
            "concreteType": "TerraformModuleVersionConnection",
            "kind": "LinkedField",
            "name": "versions",
            "plural": false,
            "selections": [
              {
                "alias": null,
                "args": null,
                "concreteType": "TerraformModuleVersionEdge",
                "kind": "LinkedField",
                "name": "edges",
                "plural": true,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "TerraformModuleVersion",
                    "kind": "LinkedField",
                    "name": "node",
                    "plural": false,
                    "selections": [
                      (v6/*: any*/)
                    ],
                    "storageKey": null
                  }
                ],
                "storageKey": null
              }
            ],
            "storageKey": null
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
      (v2/*: any*/),
      (v3/*: any*/)
    ],
    "kind": "Operation",
    "name": "ModuleVersionAutocompleteQuery",
    "selections": [
      {
        "alias": null,
        "args": (v4/*: any*/),
        "concreteType": "TerraformModule",
        "kind": "LinkedField",
        "name": "terraformModule",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": (v5/*: any*/),
            "concreteType": "TerraformModuleVersionConnection",
            "kind": "LinkedField",
            "name": "versions",
            "plural": false,
            "selections": [
              {
                "alias": null,
                "args": null,
                "concreteType": "TerraformModuleVersionEdge",
                "kind": "LinkedField",
                "name": "edges",
                "plural": true,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "TerraformModuleVersion",
                    "kind": "LinkedField",
                    "name": "node",
                    "plural": false,
                    "selections": [
                      (v6/*: any*/),
                      (v7/*: any*/)
                    ],
                    "storageKey": null
                  }
                ],
                "storageKey": null
              }
            ],
            "storageKey": null
          },
          (v7/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "c7515eeb75072c11b2c2742ccedb3042",
    "id": null,
    "metadata": {},
    "name": "ModuleVersionAutocompleteQuery",
    "operationKind": "query",
    "text": "query ModuleVersionAutocompleteQuery(\n  $registryNamespace: String!\n  $moduleName: String!\n  $system: String!\n  $versionSearch: String\n) {\n  terraformModule(registryNamespace: $registryNamespace, moduleName: $moduleName, system: $system) {\n    versions(first: 50, search: $versionSearch, sort: CREATED_AT_DESC) {\n      edges {\n        node {\n          version\n          id\n        }\n      }\n    }\n    id\n  }\n}\n"
  }
};
})();

(node as any).hash = "0c0c487d14612a94388a044b722fb7fb";

export default node;
