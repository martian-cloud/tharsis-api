/**
 * @generated SignedSource<<51f640865c8cce0fafde1411f4b83196>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type PackageVersionStatus = "ERRORED" | "PENDING" | "UPLOADED" | "UPLOAD_IN_PROGRESS" | "%future added value";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type CreatePackageVersionInput = {
  clientMutationId?: string | null | undefined;
  packageId: string;
  shaSum: string;
  version: string;
};
export type NewPackageVersionCreateMutation$variables = {
  input: CreatePackageVersionInput;
};
export type NewPackageVersionCreateMutation$data = {
  readonly createPackageVersion: {
    readonly packageVersion: {
      readonly id: string;
      readonly status: PackageVersionStatus;
      readonly version: string;
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type NewPackageVersionCreateMutation = {
  response: NewPackageVersionCreateMutation$data;
  variables: NewPackageVersionCreateMutation$variables;
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
    "concreteType": "PackageVersionMutationPayload",
    "kind": "LinkedField",
    "name": "createPackageVersion",
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
            "name": "version",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "status",
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
    "name": "NewPackageVersionCreateMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "NewPackageVersionCreateMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "16e097ee9f79f9498c93d3241b132609",
    "id": null,
    "metadata": {},
    "name": "NewPackageVersionCreateMutation",
    "operationKind": "mutation",
    "text": "mutation NewPackageVersionCreateMutation(\n  $input: CreatePackageVersionInput!\n) {\n  createPackageVersion(input: $input) {\n    packageVersion {\n      id\n      version\n      status\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "c3882b5ea8c357b078d49da62bcbd4a1";

export default node;
