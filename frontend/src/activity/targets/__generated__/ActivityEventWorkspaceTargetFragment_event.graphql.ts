/**
 * @generated SignedSource<<b4c1444593dab21c965231e26be88779>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type ActivityEventAction = "ADD" | "ADD_MEMBER" | "APPLY" | "CANCEL" | "CREATE" | "CREATE_MEMBERSHIP" | "DELETE" | "DELETE_CHILD_RESOURCE" | "LOCK" | "MIGRATE" | "REMOVE" | "REMOVE_MEMBER" | "REMOVE_MEMBERSHIP" | "SET_VARIABLES" | "UNLOCK" | "UPDATE" | "UPDATE_MEMBER" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type ActivityEventWorkspaceTargetFragment_event$data = {
  readonly action: ActivityEventAction;
  readonly payload: {
    readonly __typename: "ActivityEventCreateNamespaceMembershipPayload";
    readonly member: {
      readonly __typename: "ServiceAccount";
      readonly resourcePath: string;
    } | {
      readonly __typename: "Team";
      readonly name: string;
    } | {
      readonly __typename: "User";
      readonly username: string;
    } | {
      // This will never be '%other', but we need some
      // value in case none of the concrete values match.
      readonly __typename: "%other";
    } | null | undefined;
    readonly role: string;
  } | {
    readonly __typename: "ActivityEventCreateWorkspacePayload";
    readonly labels: ReadonlyArray<{
      readonly key: string;
      readonly value: string;
    }> | null | undefined;
  } | {
    readonly __typename: "ActivityEventDeleteChildResourcePayload";
    readonly name: string;
    readonly type: string;
  } | {
    readonly __typename: "ActivityEventMigrateWorkspacePayload";
    readonly previousGroupPath: string;
  } | {
    readonly __typename: "ActivityEventRemoveNamespaceMembershipPayload";
    readonly member: {
      readonly __typename: "ServiceAccount";
      readonly resourcePath: string;
    } | {
      readonly __typename: "Team";
      readonly name: string;
    } | {
      readonly __typename: "User";
      readonly username: string;
    } | {
      // This will never be '%other', but we need some
      // value in case none of the concrete values match.
      readonly __typename: "%other";
    } | null | undefined;
  } | {
    readonly __typename: "ActivityEventUpdateWorkspacePayload";
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
    readonly description?: string;
    readonly fullPath?: string;
    readonly name?: string;
  };
  readonly " $fragmentSpreads": FragmentRefs<"ActivityEventListItemFragment_event">;
  readonly " $fragmentType": "ActivityEventWorkspaceTargetFragment_event";
};
export type ActivityEventWorkspaceTargetFragment_event$key = {
  readonly " $data"?: ActivityEventWorkspaceTargetFragment_event$data;
  readonly " $fragmentSpreads": FragmentRefs<"ActivityEventWorkspaceTargetFragment_event">;
};

const node: ReaderFragment = (function(){
var v0 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "name",
  "storageKey": null
},
v1 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "__typename",
  "storageKey": null
},
v2 = {
  "alias": null,
  "args": null,
  "concreteType": null,
  "kind": "LinkedField",
  "name": "member",
  "plural": false,
  "selections": [
    (v1/*: any*/),
    {
      "kind": "InlineFragment",
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "username",
          "storageKey": null
        }
      ],
      "type": "User",
      "abstractKey": null
    },
    {
      "kind": "InlineFragment",
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "resourcePath",
          "storageKey": null
        }
      ],
      "type": "ServiceAccount",
      "abstractKey": null
    },
    {
      "kind": "InlineFragment",
      "selections": [
        (v0/*: any*/)
      ],
      "type": "Team",
      "abstractKey": null
    }
  ],
  "storageKey": null
},
v3 = [
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
  "name": "ActivityEventWorkspaceTargetFragment_event",
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
            (v0/*: any*/),
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "fullPath",
              "storageKey": null
            },
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "description",
              "storageKey": null
            }
          ],
          "type": "Workspace",
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
        (v1/*: any*/),
        {
          "kind": "InlineFragment",
          "selections": [
            (v0/*: any*/),
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "type",
              "storageKey": null
            }
          ],
          "type": "ActivityEventDeleteChildResourcePayload",
          "abstractKey": null
        },
        {
          "kind": "InlineFragment",
          "selections": [
            (v2/*: any*/),
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "role",
              "storageKey": null
            }
          ],
          "type": "ActivityEventCreateNamespaceMembershipPayload",
          "abstractKey": null
        },
        {
          "kind": "InlineFragment",
          "selections": [
            (v2/*: any*/)
          ],
          "type": "ActivityEventRemoveNamespaceMembershipPayload",
          "abstractKey": null
        },
        {
          "kind": "InlineFragment",
          "selections": [
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "previousGroupPath",
              "storageKey": null
            }
          ],
          "type": "ActivityEventMigrateWorkspacePayload",
          "abstractKey": null
        },
        {
          "kind": "InlineFragment",
          "selections": [
            {
              "alias": null,
              "args": null,
              "concreteType": "WorkspaceLabel",
              "kind": "LinkedField",
              "name": "labels",
              "plural": true,
              "selections": (v3/*: any*/),
              "storageKey": null
            }
          ],
          "type": "ActivityEventCreateWorkspacePayload",
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
                  "selections": (v3/*: any*/),
                  "storageKey": null
                },
                {
                  "alias": null,
                  "args": null,
                  "concreteType": "WorkspaceLabel",
                  "kind": "LinkedField",
                  "name": "updated",
                  "plural": true,
                  "selections": (v3/*: any*/),
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
          "type": "ActivityEventUpdateWorkspacePayload",
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

(node as any).hash = "b2959287fd150a10316600ae72b78101";

export default node;
