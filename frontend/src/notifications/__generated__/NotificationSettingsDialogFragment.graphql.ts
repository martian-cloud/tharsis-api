/**
 * @generated SignedSource<<db4a72829efc1a22250f534064ac5c4c>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type UserNotificationPreferenceScope = "ALL" | "CUSTOM" | "NONE" | "PARTICIPATE" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type NotificationSettingsDialogFragment$data = {
  readonly customEvents: {
    readonly failedRun: boolean;
  } | null | undefined;
  readonly inherited: boolean;
  readonly namespacePath: string | null | undefined;
  readonly scope: UserNotificationPreferenceScope;
  readonly " $fragmentType": "NotificationSettingsDialogFragment";
};
export type NotificationSettingsDialogFragment$key = {
  readonly " $data"?: NotificationSettingsDialogFragment$data;
  readonly " $fragmentSpreads": FragmentRefs<"NotificationSettingsDialogFragment">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "NotificationSettingsDialogFragment",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "scope",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "inherited",
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
      "concreteType": "UserNotificationPreferenceCustomEvents",
      "kind": "LinkedField",
      "name": "customEvents",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "failedRun",
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "UserNotificationPreference",
  "abstractKey": null
};

(node as any).hash = "4022f245fccd9602f7a80909daf6c56d";

export default node;
