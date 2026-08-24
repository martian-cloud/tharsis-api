/**
 * @generated SignedSource<<81342df097f29e003e65d31100edab99>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type DeletePackageVersionInput = {
  clientMutationId?: string | null | undefined;
  id: string;
};
export type PackageVersionActionsDeleteMutation$variables = {
  connections: ReadonlyArray<string>;
  input: DeletePackageVersionInput;
};
export type PackageVersionActionsDeleteMutation$data = {
  readonly deletePackageVersion: {
    readonly packageVersion: {
      readonly id: string;
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type PackageVersionActionsDeleteMutation = {
  response: PackageVersionActionsDeleteMutation$data;
  variables: PackageVersionActionsDeleteMutation$variables;
};

const node: ConcreteRequest = (function(){
var v0 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "connections"
},
v1 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "input"
},
v2 = [
  {
    "kind": "Variable",
    "name": "input",
    "variableName": "input"
  }
],
v3 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v4 = {
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
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "PackageVersionActionsDeleteMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "PackageVersionMutationPayload",
        "kind": "LinkedField",
        "name": "deletePackageVersion",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "PackageVersion",
            "kind": "LinkedField",
            "name": "packageVersion",
            "plural": false,
            "selections": [
              (v3/*: any*/)
            ],
            "storageKey": null
          },
          (v4/*: any*/)
        ],
        "storageKey": null
      }
    ],
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [
      (v1/*: any*/),
      (v0/*: any*/)
    ],
    "kind": "Operation",
    "name": "PackageVersionActionsDeleteMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "PackageVersionMutationPayload",
        "kind": "LinkedField",
        "name": "deletePackageVersion",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "PackageVersion",
            "kind": "LinkedField",
            "name": "packageVersion",
            "plural": false,
            "selections": [
              (v3/*: any*/),
              {
                "alias": null,
                "args": null,
                "filters": null,
                "handle": "deleteEdge",
                "key": "",
                "kind": "ScalarHandle",
                "name": "id",
                "handleArgs": [
                  {
                    "kind": "Variable",
                    "name": "connections",
                    "variableName": "connections"
                  }
                ]
              }
            ],
            "storageKey": null
          },
          (v4/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "8a6af5fbec87972657370af02d092037",
    "id": null,
    "metadata": {},
    "name": "PackageVersionActionsDeleteMutation",
    "operationKind": "mutation",
    "text": "mutation PackageVersionActionsDeleteMutation(\n  $input: DeletePackageVersionInput!\n) {\n  deletePackageVersion(input: $input) {\n    packageVersion {\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "387f54f9c05ac23d7cfdea70d8eeb23f";

export default node;
