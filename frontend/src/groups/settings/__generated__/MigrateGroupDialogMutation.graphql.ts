/**
 * @generated SignedSource<<8dfbdf26eb56713482179f0bdbb1242e>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type MigrateGroupInput = {
  clientMutationId?: string | null | undefined;
  groupId?: string | null | undefined;
  groupPath?: string | null | undefined;
  newParentId?: string | null | undefined;
  newParentPath?: string | null | undefined;
};
export type MigrateGroupDialogMutation$variables = {
  input: MigrateGroupInput;
};
export type MigrateGroupDialogMutation$data = {
  readonly migrateGroup: {
    readonly group: {
      readonly fullPath: string;
      readonly id: string;
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type MigrateGroupDialogMutation = {
  response: MigrateGroupDialogMutation$data;
  variables: MigrateGroupDialogMutation$variables;
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
    "concreteType": "MigrateGroupPayload",
    "kind": "LinkedField",
    "name": "migrateGroup",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "Group",
        "kind": "LinkedField",
        "name": "group",
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
            "name": "fullPath",
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
    "name": "MigrateGroupDialogMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "MigrateGroupDialogMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "013c7973df5679d816ae7e55fb7a69c8",
    "id": null,
    "metadata": {},
    "name": "MigrateGroupDialogMutation",
    "operationKind": "mutation",
    "text": "mutation MigrateGroupDialogMutation(\n  $input: MigrateGroupInput!\n) {\n  migrateGroup(input: $input) {\n    group {\n      id\n      fullPath\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "9a6f27c8a92860320e5e2286032bbc86";

export default node;
