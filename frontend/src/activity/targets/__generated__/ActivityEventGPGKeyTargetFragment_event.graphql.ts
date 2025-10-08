/**
 * @generated SignedSource<<ba10626b8ea37e0e742672aa1678e9d5>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type ActivityEventAction = "ADD" | "ADD_MEMBER" | "APPLY" | "CANCEL" | "CREATE" | "CREATE_MEMBERSHIP" | "DELETE" | "DELETE_CHILD_RESOURCE" | "LOCK" | "MIGRATE" | "REMOVE" | "REMOVE_MEMBER" | "REMOVE_MEMBERSHIP" | "SET_VARIABLES" | "UNLOCK" | "UPDATE" | "UPDATE_MEMBER" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type ActivityEventGPGKeyTargetFragment_event$data = {
  readonly action: ActivityEventAction;
  readonly namespacePath: string | null | undefined;
  readonly target: {
    readonly gpgKeyId?: string;
    readonly id?: string;
  };
  readonly " $fragmentSpreads": FragmentRefs<"ActivityEventListItemFragment_event">;
  readonly " $fragmentType": "ActivityEventGPGKeyTargetFragment_event";
};
export type ActivityEventGPGKeyTargetFragment_event$key = {
  readonly " $data"?: ActivityEventGPGKeyTargetFragment_event$data;
  readonly " $fragmentSpreads": FragmentRefs<"ActivityEventGPGKeyTargetFragment_event">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ActivityEventGPGKeyTargetFragment_event",
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
              "name": "gpgKeyId",
              "storageKey": null
            }
          ],
          "type": "GPGKey",
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

(node as any).hash = "40d13169ec6b8caf97711e64773f39e0";

export default node;
