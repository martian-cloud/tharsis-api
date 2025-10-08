/**
 * @generated SignedSource<<7a4ad170e50b255691187f1dac861298>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type ActivityEventAction = "ADD" | "ADD_MEMBER" | "APPLY" | "CANCEL" | "CREATE" | "CREATE_MEMBERSHIP" | "DELETE" | "DELETE_CHILD_RESOURCE" | "LOCK" | "MIGRATE" | "REMOVE" | "REMOVE_MEMBER" | "REMOVE_MEMBERSHIP" | "SET_VARIABLES" | "UNLOCK" | "UPDATE" | "UPDATE_MEMBER" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type ActivityEventRunnerTargetFragment_event$data = {
  readonly action: ActivityEventAction;
  readonly namespacePath: string | null | undefined;
  readonly target: {
    readonly name?: string;
  };
  readonly " $fragmentSpreads": FragmentRefs<"ActivityEventListItemFragment_event">;
  readonly " $fragmentType": "ActivityEventRunnerTargetFragment_event";
};
export type ActivityEventRunnerTargetFragment_event$key = {
  readonly " $data"?: ActivityEventRunnerTargetFragment_event$data;
  readonly " $fragmentSpreads": FragmentRefs<"ActivityEventRunnerTargetFragment_event">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ActivityEventRunnerTargetFragment_event",
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
              "name": "name",
              "storageKey": null
            }
          ],
          "type": "Runner",
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

(node as any).hash = "8bac5de9edb87007775444f3b163c5d1";

export default node;
