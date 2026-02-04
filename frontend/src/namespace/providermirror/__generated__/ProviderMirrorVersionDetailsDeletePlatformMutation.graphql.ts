/**
 * @generated SignedSource<<0b2f311dee8c32b2a6917a15a495aa3f>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type DeleteTerraformProviderPlatformMirrorInput = {
  clientMutationId?: string | null | undefined;
  id: string;
  metadata?: ResourceMetadataInput | null | undefined;
};
export type ResourceMetadataInput = {
  version: string;
};
export type ProviderMirrorVersionDetailsDeletePlatformMutation$variables = {
  input: DeleteTerraformProviderPlatformMirrorInput;
};
export type ProviderMirrorVersionDetailsDeletePlatformMutation$data = {
  readonly deleteTerraformProviderPlatformMirror: {
    readonly problems: ReadonlyArray<{
      readonly message: string;
    }>;
  };
};
export type ProviderMirrorVersionDetailsDeletePlatformMutation = {
  response: ProviderMirrorVersionDetailsDeletePlatformMutation$data;
  variables: ProviderMirrorVersionDetailsDeletePlatformMutation$variables;
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
    "concreteType": "DeleteTerraformProviderPlatformMirrorPayload",
    "kind": "LinkedField",
    "name": "deleteTerraformProviderPlatformMirror",
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
    "name": "ProviderMirrorVersionDetailsDeletePlatformMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "ProviderMirrorVersionDetailsDeletePlatformMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "367af68adb10b56281e458aa9ca48758",
    "id": null,
    "metadata": {},
    "name": "ProviderMirrorVersionDetailsDeletePlatformMutation",
    "operationKind": "mutation",
    "text": "mutation ProviderMirrorVersionDetailsDeletePlatformMutation(\n  $input: DeleteTerraformProviderPlatformMirrorInput!\n) {\n  deleteTerraformProviderPlatformMirror(input: $input) {\n    problems {\n      message\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "d191d4adbef8f939f45715f1c3579e7e";

export default node;
