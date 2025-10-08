/**
 * @generated SignedSource<<94d9c2267502d5fd80a06a81bf0b7a2a>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type ActivityEventAction = "ADD" | "ADD_MEMBER" | "APPLY" | "CANCEL" | "CREATE" | "CREATE_MEMBERSHIP" | "DELETE" | "DELETE_CHILD_RESOURCE" | "LOCK" | "MIGRATE" | "REMOVE" | "REMOVE_MEMBER" | "REMOVE_MEMBERSHIP" | "SET_VARIABLES" | "UNLOCK" | "UPDATE" | "UPDATE_MEMBER" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type ActivityEventFederatedRegistryTargetFragment_event$data = {
  readonly action: ActivityEventAction;
  readonly namespacePath: string | null | undefined;
  readonly target: {
    readonly hostname?: string;
    readonly id?: string;
  };
  readonly " $fragmentSpreads": FragmentRefs<"ActivityEventListItemFragment_event">;
  readonly " $fragmentType": "ActivityEventFederatedRegistryTargetFragment_event";
};
export type ActivityEventFederatedRegistryTargetFragment_event$key = {
  readonly " $data"?: ActivityEventFederatedRegistryTargetFragment_event$data;
  readonly " $fragmentSpreads": FragmentRefs<"ActivityEventFederatedRegistryTargetFragment_event">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ActivityEventFederatedRegistryTargetFragment_event",
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
            },
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "hostname",
              "storageKey": null
            }
          ],
          "type": "FederatedRegistry",
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

(node as any).hash = "2ebb370d4612491a74d2ebbafebd17f6";

export default node;
