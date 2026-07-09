/**
 * @generated SignedSource<<fca19c00b9a77e42604b1e99418df11d>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type CheckResultStatus = "error" | "fail" | "pass" | "unknown" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type CheckResultsPanelFragment_checkResult$data = ReadonlyArray<{
  readonly name: string;
  readonly objects: ReadonlyArray<{
    readonly address: string;
    readonly failureMessages: ReadonlyArray<string>;
    readonly status: CheckResultStatus;
  }>;
  readonly status: CheckResultStatus;
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
      "concreteType": "CheckResultObject",
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
  "type": "CheckResult",
  "abstractKey": null
};
})();

(node as any).hash = "96e6e5fd097b672a2f0bc652caef6880";

export default node;
