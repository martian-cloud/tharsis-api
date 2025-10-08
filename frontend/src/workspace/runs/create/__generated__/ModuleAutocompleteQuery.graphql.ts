/**
 * @generated SignedSource<<9de77ec83337086c95c07aa027b82f5e>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ModuleAutocompleteQuery$variables = {
  search: string;
};
export type ModuleAutocompleteQuery$data = {
  readonly terraformModules: {
    readonly edges: ReadonlyArray<{
      readonly node: {
        readonly groupPath: string;
        readonly id: string;
        readonly name: string;
        readonly private: boolean;
        readonly registryNamespace: string;
        readonly resourcePath: string;
        readonly source: string;
        readonly system: string;
      } | null | undefined;
    } | null | undefined> | null | undefined;
  };
};
export type ModuleAutocompleteQuery = {
  response: ModuleAutocompleteQuery$data;
  variables: ModuleAutocompleteQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "search"
  }
],
v1 = [
  {
    "alias": null,
    "args": [
      {
        "kind": "Literal",
        "name": "first",
        "value": 50
      },
      {
        "kind": "Variable",
        "name": "search",
        "variableName": "search"
      }
    ],
    "concreteType": "TerraformModuleConnection",
    "kind": "LinkedField",
    "name": "terraformModules",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "TerraformModuleEdge",
        "kind": "LinkedField",
        "name": "edges",
        "plural": true,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "TerraformModule",
            "kind": "LinkedField",
            "name": "node",
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
                "name": "source",
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
                "name": "resourcePath",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "groupPath",
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
                "name": "system",
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
];
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "ModuleAutocompleteQuery",
    "selections": (v1/*: any*/),
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "ModuleAutocompleteQuery",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "83b682780f0b404305d72bb441179521",
    "id": null,
    "metadata": {},
    "name": "ModuleAutocompleteQuery",
    "operationKind": "query",
    "text": "query ModuleAutocompleteQuery(\n  $search: String!\n) {\n  terraformModules(first: 50, search: $search) {\n    edges {\n      node {\n        id\n        name\n        source\n        private\n        resourcePath\n        groupPath\n        registryNamespace\n        system\n      }\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "b97c07a40f4108b92041b6c89b018231";

export default node;
