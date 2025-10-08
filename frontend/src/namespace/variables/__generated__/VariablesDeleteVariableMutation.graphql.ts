/**
 * @generated SignedSource<<2e10127c95a657b82199fd6a3cea40f3>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type DeleteNamespaceVariableInput = {
  clientMutationId?: string | null | undefined;
  id: string;
};
export type VariablesDeleteVariableMutation$variables = {
  input: DeleteNamespaceVariableInput;
};
export type VariablesDeleteVariableMutation$data = {
  readonly deleteNamespaceVariable: {
    readonly namespace: {
      readonly id: string;
      readonly variables: ReadonlyArray<{
        readonly " $fragmentSpreads": FragmentRefs<"VariableListItemFragment_variable">;
      }>;
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type VariablesDeleteVariableMutation = {
  response: VariablesDeleteVariableMutation$data;
  variables: VariablesDeleteVariableMutation$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "input"
  }
],
v1 = [
  {
    "kind": "Variable",
    "name": "input",
    "variableName": "input"
  }
],
v2 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v3 = {
  "alias": null,
  "args": null,
  "concreteType": "Problem",
  "kind": "LinkedField",
  "name": "problems",
  "plural": true,
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "message",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "field",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "type",
      "storageKey": null
    }
  ],
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "VariablesDeleteVariableMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "NamespaceVariableMutationPayload",
        "kind": "LinkedField",
        "name": "deleteNamespaceVariable",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": null,
            "kind": "LinkedField",
            "name": "namespace",
            "plural": false,
            "selections": [
              (v2/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": "NamespaceVariable",
                "kind": "LinkedField",
                "name": "variables",
                "plural": true,
                "selections": [
                  {
                    "args": null,
                    "kind": "FragmentSpread",
                    "name": "VariableListItemFragment_variable"
                  }
                ],
                "storageKey": null
              }
            ],
            "storageKey": null
          },
          (v3/*: any*/)
        ],
        "storageKey": null
      }
    ],
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "VariablesDeleteVariableMutation",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": "NamespaceVariableMutationPayload",
        "kind": "LinkedField",
        "name": "deleteNamespaceVariable",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": null,
            "kind": "LinkedField",
            "name": "namespace",
            "plural": false,
            "selections": [
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "__typename",
                "storageKey": null
              },
              (v2/*: any*/),
              {
                "alias": null,
                "args": null,
                "concreteType": "NamespaceVariable",
                "kind": "LinkedField",
                "name": "variables",
                "plural": true,
                "selections": [
                  (v2/*: any*/),
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
                    "name": "category",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "sensitive",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "value",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "namespacePath",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "latestVersionId",
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
                        "name": "updatedAt",
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
          },
          (v3/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "b920450742c42da5c168f6274ac4e44d",
    "id": null,
    "metadata": {},
    "name": "VariablesDeleteVariableMutation",
    "operationKind": "mutation",
    "text": "mutation VariablesDeleteVariableMutation(\n  $input: DeleteNamespaceVariableInput!\n) {\n  deleteNamespaceVariable(input: $input) {\n    namespace {\n      __typename\n      id\n      variables {\n        ...VariableListItemFragment_variable\n        id\n      }\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment VariableListItemFragment_variable on NamespaceVariable {\n  id\n  key\n  category\n  sensitive\n  value\n  namespacePath\n  latestVersionId\n  metadata {\n    updatedAt\n  }\n}\n"
  }
};
})();

(node as any).hash = "c626ac23845ae2b4ea980060205b0c10";

export default node;
