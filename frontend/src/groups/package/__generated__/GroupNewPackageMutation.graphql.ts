/**
 * @generated SignedSource<<2985cde20d9482a7d4624caf55cfcd7c>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type PackageKind = "OPA_POLICY" | "%future added value";
export type PackageVisibility = "GLOBAL" | "PRIVATE" | "ROOT_GROUP" | "%future added value";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type CreatePackageInput = {
  allowMutableVersions?: boolean | null | undefined;
  clientMutationId?: string | null | undefined;
  description?: string | null | undefined;
  groupId: string;
  kind: PackageKind;
  name: string;
  visibility: PackageVisibility;
};
export type GroupNewPackageMutation$variables = {
  input: CreatePackageInput;
};
export type GroupNewPackageMutation$data = {
  readonly createPackage: {
    readonly package: {
      readonly allowMutableVersions: boolean;
      readonly description: string;
      readonly id: string;
      readonly name: string;
      readonly visibility: PackageVisibility;
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type GroupNewPackageMutation = {
  response: GroupNewPackageMutation$data;
  variables: GroupNewPackageMutation$variables;
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
    "name": "createPackage",
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
            "name": "name",
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
    "name": "GroupNewPackageMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "GroupNewPackageMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "9ed4d7debdd8b7c64417b08032fdf23a",
    "id": null,
    "metadata": {},
    "name": "GroupNewPackageMutation",
    "operationKind": "mutation",
    "text": "mutation GroupNewPackageMutation(\n  $input: CreatePackageInput!\n) {\n  createPackage(input: $input) {\n    package {\n      id\n      name\n      description\n      visibility\n      allowMutableVersions\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "a2f3f7fba3235caf4c98f098e1975c44";

export default node;
