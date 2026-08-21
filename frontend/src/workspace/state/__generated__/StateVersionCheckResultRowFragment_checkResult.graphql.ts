/**
 * @generated SignedSource<<d2c386948caac1eff1cc9b5edd1a5adf>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type TerraformCheckResultStatus = "ERROR" | "FAIL" | "PASS" | "UNKNOWN" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type StateVersionCheckResultRowFragment_checkResult$data = {
  readonly name: string;
  readonly objects: ReadonlyArray<{
    readonly address: string;
    readonly failureMessages: ReadonlyArray<string>;
    readonly status: TerraformCheckResultStatus;
  }>;
  readonly status: TerraformCheckResultStatus;
  readonly " $fragmentType": "StateVersionCheckResultRowFragment_checkResult";
};
export type StateVersionCheckResultRowFragment_checkResult$key = {
  readonly " $data"?: StateVersionCheckResultRowFragment_checkResult$data;
  readonly " $fragmentSpreads": FragmentRefs<"StateVersionCheckResultRowFragment_checkResult">;
};

const node: ReaderFragment = (function(){
var v0 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "status",
  "storageKey": null
};
return {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "StateVersionCheckResultRowFragment_checkResult",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "name",
      "storageKey": null
    },
    (v0/*: any*/),
    {
      "alias": null,
      "args": null,
      "concreteType": "TerraformCheckResultObject",
      "kind": "LinkedField",
      "name": "objects",
      "plural": true,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "address",
          "storageKey": null
        },
        (v0/*: any*/),
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "failureMessages",
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "TerraformCheckResult",
  "abstractKey": null
};
})();

(node as any).hash = "7dbbd3a105a3a0beb57c528c28656134";

export default node;
