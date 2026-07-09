/**
 * @generated SignedSource<<09c35abb2f49026bb1add787f1074d7a>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type CheckResultStatus = "error" | "fail" | "pass" | "unknown" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type StateVersionCheckResultRowFragment_checkResult$data = {
  readonly name: string;
  readonly objects: ReadonlyArray<{
    readonly address: string;
    readonly failureMessages: ReadonlyArray<string>;
    readonly status: CheckResultStatus;
  }>;
  readonly status: CheckResultStatus;
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

(node as any).hash = "513b6f122e6ef5e42b9567d27d939e08";

export default node;
