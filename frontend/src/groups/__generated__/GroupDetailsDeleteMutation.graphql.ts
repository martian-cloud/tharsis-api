/**
 * @generated SignedSource<<0be963f13515e6e78a041ca950e6f227>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type DeleteGroupInput = {
  clientMutationId?: string | null | undefined;
  force?: boolean | null | undefined;
  groupPath?: string | null | undefined;
  id?: string | null | undefined;
  metadata?: ResourceMetadataInput | null | undefined;
};
export type ResourceMetadataInput = {
  version: string;
};
export type GroupDetailsDeleteMutation$variables = {
  input: DeleteGroupInput;
};
export type GroupDetailsDeleteMutation$data = {
  readonly deleteGroup: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type GroupDetailsDeleteMutation = {
  response: GroupDetailsDeleteMutation$data;
  variables: GroupDetailsDeleteMutation$variables;
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
    "concreteType": "DeleteGroupPayload",
    "kind": "LinkedField",
    "name": "deleteGroup",
    "plural": false,
    "selections": [
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
    "name": "GroupDetailsDeleteMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "GroupDetailsDeleteMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "2a686d578a5b7648c4ec5b0a8a83f80d",
    "id": null,
    "metadata": {},
    "name": "GroupDetailsDeleteMutation",
    "operationKind": "mutation",
    "text": "mutation GroupDetailsDeleteMutation(\n  $input: DeleteGroupInput!\n) {\n  deleteGroup(input: $input) {\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "5aea3dfbc1afd6d35f10f0dc3cab74fe";

export default node;
