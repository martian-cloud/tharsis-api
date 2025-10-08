/**
 * @generated SignedSource<<b95afbfed9c6f672049901412f3a893b>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type UpdateVCSProviderInput = {
  clientMutationId?: string | null | undefined;
  description?: string | null | undefined;
  id: string;
  metadata?: ResourceMetadataInput | null | undefined;
  oAuthClientId?: string | null | undefined;
  oAuthClientSecret?: string | null | undefined;
};
export type ResourceMetadataInput = {
  version: string;
};
export type EditVCSProviderMutation$variables = {
  input: UpdateVCSProviderInput;
};
export type EditVCSProviderMutation$data = {
  readonly updateVCSProvider: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly vcsProvider: {
      readonly description: string;
      readonly id: string;
    } | null | undefined;
  };
};
export type EditVCSProviderMutation = {
  response: EditVCSProviderMutation$data;
  variables: EditVCSProviderMutation$variables;
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
    "concreteType": "UpdateVCSProviderPayload",
    "kind": "LinkedField",
    "name": "updateVCSProvider",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "VCSProvider",
        "kind": "LinkedField",
        "name": "vcsProvider",
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
    "name": "EditVCSProviderMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "EditVCSProviderMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "bc154fd403dc79565b3af2400fcb3524",
    "id": null,
    "metadata": {},
    "name": "EditVCSProviderMutation",
    "operationKind": "mutation",
    "text": "mutation EditVCSProviderMutation(\n  $input: UpdateVCSProviderInput!\n) {\n  updateVCSProvider(input: $input) {\n    vcsProvider {\n      id\n      description\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "cff18271cffe7c7361283351f963d15a";

export default node;
