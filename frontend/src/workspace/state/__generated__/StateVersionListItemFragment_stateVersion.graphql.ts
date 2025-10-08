/**
 * @generated SignedSource<<fd37bc12332d82d0ddafd47721083bc0>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type StateVersionListItemFragment_stateVersion$data = {
  readonly createdBy: string;
  readonly id: string;
  readonly metadata: {
    readonly createdAt: any;
    readonly trn: string;
  };
  readonly run: {
    readonly createdBy: string;
    readonly id: string;
  } | null | undefined;
  readonly " $fragmentType": "StateVersionListItemFragment_stateVersion";
};
export type StateVersionListItemFragment_stateVersion$key = {
  readonly " $data"?: StateVersionListItemFragment_stateVersion$data;
  readonly " $fragmentSpreads": FragmentRefs<"StateVersionListItemFragment_stateVersion">;
};

const node: ReaderFragment = (function(){
var v0 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v1 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "createdBy",
  "storageKey": null
};
return {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "StateVersionListItemFragment_stateVersion",
  "selections": [
    (v0/*: any*/),
    (v1/*: any*/),
    {
      "alias": null,
      "args": null,
      "concreteType": "ResourceMetadata",
      "kind": "LinkedField",
      "name": "metadata",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "createdAt",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "trn",
          "storageKey": null
        }
      ],
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "Run",
      "kind": "LinkedField",
      "name": "run",
      "plural": false,
      "selections": [
        (v0/*: any*/),
        (v1/*: any*/)
      ],
      "storageKey": null
    }
  ],
  "type": "StateVersion",
  "abstractKey": null
};
})();

(node as any).hash = "85b5a6b9518a556bdb8c47bd265069f7";

export default node;
