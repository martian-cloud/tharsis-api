/**
 * @generated SignedSource<<9dac1019820a69190171f7cb6dd9b4d7>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type DeletePackageInput = {
  clientMutationId?: string | null | undefined;
  force?: boolean | null | undefined;
  id: string;
};
export type GroupEditPackageDeleteMutation$variables = {
  input: DeletePackageInput;
};
export type GroupEditPackageDeleteMutation$data = {
  readonly deletePackage: {
    readonly package: {
      readonly id: string;
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type GroupEditPackageDeleteMutation = {
  response: GroupEditPackageDeleteMutation$data;
  variables: GroupEditPackageDeleteMutation$variables;
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
    "alias": null,
    "args": [
      {
        "kind": "Variable",
        "name": "input",
        "variableName": "input"
      }
    ],
    "concreteType": "PackageMutationPayload",
    "kind": "LinkedField",
    "name": "deletePackage",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "Package",
        "kind": "LinkedField",
        "name": "package",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "id",
            "storageKey": null
          }
        ],
        "storageKey": null
      },
      {
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
    "name": "GroupEditPackageDeleteMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "GroupEditPackageDeleteMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "b834b2158daac259ee06075c66af2dc5",
    "id": null,
    "metadata": {},
    "name": "GroupEditPackageDeleteMutation",
    "operationKind": "mutation",
    "text": "mutation GroupEditPackageDeleteMutation(\n  $input: DeletePackageInput!\n) {\n  deletePackage(input: $input) {\n    package {\n      id\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "4b8e2728f918607c3d7777fd545fc9a5";

export default node;
