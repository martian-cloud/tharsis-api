/**
 * @generated SignedSource<<53c3df5221a8c755f2d8504377515a81>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type PackageVisibility = "GLOBAL" | "PRIVATE" | "ROOT_GROUP" | "%future added value";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type UpdatePackageInput = {
  allowMutableVersions?: boolean | null | undefined;
  clientMutationId?: string | null | undefined;
  description?: string | null | undefined;
  id: string;
  visibility?: PackageVisibility | null | undefined;
};
export type GroupEditPackageMutation$variables = {
  input: UpdatePackageInput;
};
export type GroupEditPackageMutation$data = {
  readonly updatePackage: {
    readonly package: {
      readonly allowMutableVersions: boolean;
      readonly description: string;
      readonly id: string;
      readonly visibility: PackageVisibility;
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type GroupEditPackageMutation = {
  response: GroupEditPackageMutation$data;
  variables: GroupEditPackageMutation$variables;
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
    "name": "updatePackage",
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
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "description",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "visibility",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "allowMutableVersions",
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
    "name": "GroupEditPackageMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "GroupEditPackageMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "27c56bd028459b03da5019b0d912754a",
    "id": null,
    "metadata": {},
    "name": "GroupEditPackageMutation",
    "operationKind": "mutation",
    "text": "mutation GroupEditPackageMutation(\n  $input: UpdatePackageInput!\n) {\n  updatePackage(input: $input) {\n    package {\n      id\n      description\n      visibility\n      allowMutableVersions\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "844679122a4dadf4bbf9f8279c954997";

export default node;
