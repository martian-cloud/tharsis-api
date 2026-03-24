/**
 * @generated SignedSource<<68a659b03f34affcb8153261b014ac57>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type ActivityEventAction = "ADD" | "ADD_MEMBER" | "APPLY" | "CANCEL" | "CREATE" | "CREATE_MEMBERSHIP" | "DELETE" | "DELETE_CHILD_RESOURCE" | "LOCK" | "MIGRATE" | "REMOVE" | "REMOVE_MEMBER" | "REMOVE_MEMBERSHIP" | "SET_VARIABLES" | "UNLOCK" | "UPDATE" | "UPDATE_MEMBER" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type ActivityEventTerraformModuleTargetFragment_event$data = {
  readonly action: ActivityEventAction;
  readonly namespacePath: string | null | undefined;
  readonly payload: {
    readonly __typename: "ActivityEventCreateTerraformModulePayload";
    readonly labels: ReadonlyArray<{
      readonly key: string;
      readonly value: string;
    }> | null | undefined;
  } | {
    readonly __typename: "ActivityEventUpdateTerraformModulePayload";
    readonly labelChanges: {
      readonly added: ReadonlyArray<{
        readonly key: string;
        readonly value: string;
      }> | null | undefined;
      readonly removed: ReadonlyArray<string> | null | undefined;
      readonly updated: ReadonlyArray<{
        readonly key: string;
        readonly value: string;
      }> | null | undefined;
    } | null | undefined;
  } | {
    // This will never be '%other', but we need some
    // value in case none of the concrete values match.
    readonly __typename: "%other";
  } | null | undefined;
  readonly target: {
    readonly name?: string;
    readonly registryNamespace?: string;
    readonly system?: string;
  };
  readonly " $fragmentSpreads": FragmentRefs<"ActivityEventListItemFragment_event">;
  readonly " $fragmentType": "ActivityEventTerraformModuleTargetFragment_event";
};
export type ActivityEventTerraformModuleTargetFragment_event$key = {
  readonly " $data"?: ActivityEventTerraformModuleTargetFragment_event$data;
  readonly " $fragmentSpreads": FragmentRefs<"ActivityEventTerraformModuleTargetFragment_event">;
};

const node: ReaderFragment = (function(){
var v0 = [
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "key",
    "storageKey": null
  },
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "value",
    "storageKey": null
  }
];
return {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ActivityEventTerraformModuleTargetFragment_event",
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
            },
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "system",
              "storageKey": null
            },
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "registryNamespace",
              "storageKey": null
            }
          ],
          "type": "TerraformModule",
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
              "alias": null,
              "args": null,
              "concreteType": "TerraformModuleLabel",
              "kind": "LinkedField",
              "name": "labels",
              "plural": true,
              "selections": (v0/*: any*/),
              "storageKey": null
            }
          ],
          "type": "ActivityEventCreateTerraformModulePayload",
          "abstractKey": null
        },
        {
          "kind": "InlineFragment",
          "selections": [
            {
              "alias": null,
              "args": null,
              "concreteType": "LabelChangePayload",
              "kind": "LinkedField",
              "name": "labelChanges",
              "plural": false,
              "selections": [
                {
                  "alias": null,
                  "args": null,
                  "concreteType": "WorkspaceLabel",
                  "kind": "LinkedField",
                  "name": "added",
                  "plural": true,
                  "selections": (v0/*: any*/),
                  "storageKey": null
                },
                {
                  "alias": null,
                  "args": null,
                  "concreteType": "WorkspaceLabel",
                  "kind": "LinkedField",
                  "name": "updated",
                  "plural": true,
                  "selections": (v0/*: any*/),
                  "storageKey": null
                },
                {
                  "alias": null,
                  "args": null,
                  "kind": "ScalarField",
                  "name": "removed",
                  "storageKey": null
                }
              ],
              "storageKey": null
            }
          ],
          "type": "ActivityEventUpdateTerraformModulePayload",
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

(node as any).hash = "eec0df3a5e56ddb84f05d70de6353c57";

export default node;
