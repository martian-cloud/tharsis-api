/**
 * @generated SignedSource<<87aa05caf99db43b2a65841177ee464b>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type ActivityEventAction = "ADD" | "ADD_MEMBER" | "APPLY" | "CANCEL" | "CREATE" | "CREATE_MEMBERSHIP" | "DELETE" | "DELETE_CHILD_RESOURCE" | "LOCK" | "MIGRATE" | "REMOVE" | "REMOVE_MEMBER" | "REMOVE_MEMBERSHIP" | "SET_VARIABLES" | "UNLOCK" | "UPDATE" | "UPDATE_MEMBER" | "%future added value";
export type JobType = "apply" | "plan" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type ActivityEventManagedIdentityAccessRuleTargetFragment_event$data = {
  readonly action: ActivityEventAction;
  readonly namespacePath: string | null | undefined;
  readonly target: {
    readonly __typename: "ManagedIdentityAccessRule";
    readonly managedIdentity: {
      readonly id: string;
      readonly resourcePath: string;
    };
    readonly runStage: JobType;
  } | {
    // This will never be '%other', but we need some
    // value in case none of the concrete values match.
    readonly __typename: "%other";
  };
  readonly " $fragmentSpreads": FragmentRefs<"ActivityEventListItemFragment_event">;
  readonly " $fragmentType": "ActivityEventManagedIdentityAccessRuleTargetFragment_event";
};
export type ActivityEventManagedIdentityAccessRuleTargetFragment_event$key = {
  readonly " $data"?: ActivityEventManagedIdentityAccessRuleTargetFragment_event$data;
  readonly " $fragmentSpreads": FragmentRefs<"ActivityEventManagedIdentityAccessRuleTargetFragment_event">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ActivityEventManagedIdentityAccessRuleTargetFragment_event",
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
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "runStage",
              "storageKey": null
            },
            {
              "alias": null,
              "args": null,
              "concreteType": "ManagedIdentity",
              "kind": "LinkedField",
              "name": "managedIdentity",
              "plural": false,
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
                  "name": "resourcePath",
                  "storageKey": null
                }
              ],
              "storageKey": null
            }
          ],
          "type": "ManagedIdentityAccessRule",
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

(node as any).hash = "16a7019c9b366da97f007f82d22f3284";

export default node;
