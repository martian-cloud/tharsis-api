/**
 * @generated SignedSource<<344a1fc856399ec9617347488d143b73>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type TerraformCheckResultStatus = "ERROR" | "FAIL" | "PASS" | "UNKNOWN" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type CheckResultsPanelFragment_checkResult$data = ReadonlyArray<{
  readonly name: string;
  readonly objects: ReadonlyArray<{
    readonly address: string;
    readonly failureMessages: ReadonlyArray<string>;
    readonly status: TerraformCheckResultStatus;
  }>;
  readonly status: TerraformCheckResultStatus;
  readonly " $fragmentType": "CheckResultsPanelFragment_checkResult";
}>;
export type CheckResultsPanelFragment_checkResult$key = ReadonlyArray<{
  readonly " $data"?: CheckResultsPanelFragment_checkResult$data;
  readonly " $fragmentSpreads": FragmentRefs<"CheckResultsPanelFragment_checkResult">;
}>;

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
  "metadata": {
    "plural": true
  },
  "name": "CheckResultsPanelFragment_checkResult",
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

(node as any).hash = "dfed4b6a8e6151baa838ee5908995d96";

export default node;
