/**
 * @generated SignedSource<<e1c9d0fe9d5901526fcf2628eba0e6ec>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type ActivityEventAction = "ADD" | "ADD_MEMBER" | "APPLY" | "CANCEL" | "CREATE" | "CREATE_MEMBERSHIP" | "DELETE" | "DELETE_CHILD_RESOURCE" | "LOCK" | "MIGRATE" | "REMOVE" | "REMOVE_MEMBER" | "REMOVE_MEMBERSHIP" | "SET_VARIABLES" | "UNLOCK" | "UPDATE" | "UPDATE_MEMBER" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type ActivityEventRunTargetFragment_event$data = {
  readonly action: ActivityEventAction;
  readonly namespacePath: string | null | undefined;
  readonly target: {
    readonly id?: string;
  };
  readonly " $fragmentSpreads": FragmentRefs<"ActivityEventListItemFragment_event">;
  readonly " $fragmentType": "ActivityEventRunTargetFragment_event";
};
export type ActivityEventRunTargetFragment_event$key = {
  readonly " $data"?: ActivityEventRunTargetFragment_event$data;
  readonly " $fragmentSpreads": FragmentRefs<"ActivityEventRunTargetFragment_event">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ActivityEventRunTargetFragment_event",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "action",
      "storageKey": null
    },
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
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "id",
              "storageKey": null
            }
          ],
          "type": "Run",
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

(node as any).hash = "9cd0d6e7604bdf0070add7b96a151356";

export default node;
