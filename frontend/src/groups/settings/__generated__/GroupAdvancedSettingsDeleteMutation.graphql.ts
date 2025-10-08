/**
 * @generated SignedSource<<b7c5b5cf756af052a50b611fa332671f>>
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
export type GroupAdvancedSettingsDeleteMutation$variables = {
  input: DeleteGroupInput;
};
export type GroupAdvancedSettingsDeleteMutation$data = {
  readonly deleteGroup: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type GroupAdvancedSettingsDeleteMutation = {
  response: GroupAdvancedSettingsDeleteMutation$data;
  variables: GroupAdvancedSettingsDeleteMutation$variables;
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
    "name": "GroupAdvancedSettingsDeleteMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "GroupAdvancedSettingsDeleteMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "963b6659e2bcc751fc1e50bee3ee61e8",
    "id": null,
    "metadata": {},
    "name": "GroupAdvancedSettingsDeleteMutation",
    "operationKind": "mutation",
    "text": "mutation GroupAdvancedSettingsDeleteMutation(\n  $input: DeleteGroupInput!\n) {\n  deleteGroup(input: $input) {\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "9655f190db3ec5db0a924a27f563156b";

export default node;
