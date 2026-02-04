/**
 * @generated SignedSource<<cea48e195b733c66edb2fae523813376>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type DeleteTerraformProviderVersionMirrorInput = {
  clientMutationId?: string | null | undefined;
  force?: boolean | null | undefined;
  id: string;
  metadata?: ResourceMetadataInput | null | undefined;
};
export type ResourceMetadataInput = {
  version: string;
};
export type ProviderMirrorVersionDetailsDeleteMutation$variables = {
  input: DeleteTerraformProviderVersionMirrorInput;
};
export type ProviderMirrorVersionDetailsDeleteMutation$data = {
  readonly deleteTerraformProviderVersionMirror: {
    readonly problems: ReadonlyArray<{
      readonly message: string;
    }>;
  };
};
export type ProviderMirrorVersionDetailsDeleteMutation = {
  response: ProviderMirrorVersionDetailsDeleteMutation$data;
  variables: ProviderMirrorVersionDetailsDeleteMutation$variables;
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
    "concreteType": "DeleteTerraformProviderVersionMirrorPayload",
    "kind": "LinkedField",
    "name": "deleteTerraformProviderVersionMirror",
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
    "name": "ProviderMirrorVersionDetailsDeleteMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "ProviderMirrorVersionDetailsDeleteMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "62abcc72c242e1bef3fec20cda5648ef",
    "id": null,
    "metadata": {},
    "name": "ProviderMirrorVersionDetailsDeleteMutation",
    "operationKind": "mutation",
    "text": "mutation ProviderMirrorVersionDetailsDeleteMutation(\n  $input: DeleteTerraformProviderVersionMirrorInput!\n) {\n  deleteTerraformProviderVersionMirror(input: $input) {\n    problems {\n      message\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "c3c326e91ff125511ceefaef94c0ca64";

export default node;
