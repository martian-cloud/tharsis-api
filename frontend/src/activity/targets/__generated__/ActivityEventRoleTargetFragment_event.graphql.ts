/**
 * @generated SignedSource<<98a236aa65742e60c93b7bb12f921e26>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type ActivityEventAction = "ADD" | "ADD_MEMBER" | "APPLY" | "CANCEL" | "CREATE" | "CREATE_MEMBERSHIP" | "DELETE" | "DELETE_CHILD_RESOURCE" | "LOCK" | "MIGRATE" | "REMOVE" | "REMOVE_MEMBER" | "REMOVE_MEMBERSHIP" | "SET_VARIABLES" | "UNLOCK" | "UPDATE" | "UPDATE_MEMBER" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type ActivityEventRoleTargetFragment_event$data = {
  readonly action: ActivityEventAction;
  readonly target: {
    readonly name?: string;
  };
  readonly " $fragmentSpreads": FragmentRefs<"ActivityEventListItemFragment_event">;
  readonly " $fragmentType": "ActivityEventRoleTargetFragment_event";
};
export type ActivityEventRoleTargetFragment_event$key = {
  readonly " $data"?: ActivityEventRoleTargetFragment_event$data;
  readonly " $fragmentSpreads": FragmentRefs<"ActivityEventRoleTargetFragment_event">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ActivityEventRoleTargetFragment_event",
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
          "type": "Role",
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

(node as any).hash = "f6e3da474c8e60354c3189ae3c9e965c";

export default node;
