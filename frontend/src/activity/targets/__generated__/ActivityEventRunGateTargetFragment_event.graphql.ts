/**
 * @generated SignedSource<<a21a0af53500d1d56f52ffa60120f005>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type ActivityEventUpdateRunGatePayloadType = "APPROVE" | "OVERRIDE" | "REJECT" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type ActivityEventRunGateTargetFragment_event$data = {
  readonly namespacePath: string | null | undefined;
  readonly payload: {
    readonly __typename: "ActivityEventUpdateRunGatePayload";
    readonly comment: string | null | undefined;
    readonly gateUpdateType: ActivityEventUpdateRunGatePayloadType;
  } | {
    // This will never be '%other', but we need some
    // value in case none of the concrete values match.
    readonly __typename: "%other";
  } | null | undefined;
  readonly target: {
    readonly id?: string;
    readonly run?: {
      readonly id: string;
    };
  } | null | undefined;
  readonly " $fragmentSpreads": FragmentRefs<"ActivityEventListItemFragment_event">;
  readonly " $fragmentType": "ActivityEventRunGateTargetFragment_event";
};
export type ActivityEventRunGateTargetFragment_event$key = {
  readonly " $data"?: ActivityEventRunGateTargetFragment_event$data;
  readonly " $fragmentSpreads": FragmentRefs<"ActivityEventRunGateTargetFragment_event">;
};

const node: ReaderFragment = (function(){
var v0 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
};
return {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ActivityEventRunGateTargetFragment_event",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "namespacePath",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": null,
      "kind": "LinkedField",
      "name": "target",
      "plural": false,
      "selections": [
        {
          "kind": "InlineFragment",
          "selections": [
            (v0/*: any*/),
            {
              "alias": null,
              "args": null,
              "concreteType": "Run",
              "kind": "LinkedField",
              "name": "run",
              "plural": false,
              "selections": [
                (v0/*: any*/)
              ],
              "storageKey": null
            }
          ],
          "type": "RunGate",
          "abstractKey": null
        }
      ],
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": null,
      "kind": "LinkedField",
      "name": "payload",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "__typename",
          "storageKey": null
        },
        {
          "kind": "InlineFragment",
          "selections": [
            {
              "alias": "gateUpdateType",
              "args": null,
              "kind": "ScalarField",
              "name": "type",
              "storageKey": null
            },
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "comment",
              "storageKey": null
            }
          ],
          "type": "ActivityEventUpdateRunGatePayload",
          "abstractKey": null
        }
      ],
      "storageKey": null
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "ActivityEventListItemFragment_event"
    }
  ],
  "type": "ActivityEvent",
  "abstractKey": null
};
})();

(node as any).hash = "c2e910d23b08975fd660cc33a6f58e7f";

export default node;
