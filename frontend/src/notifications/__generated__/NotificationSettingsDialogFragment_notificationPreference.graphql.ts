/**
 * @generated SignedSource<<2f80883e8ec7298818645282c3c3f0f0>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type UserNotificationPreferenceScope = "ALL" | "CUSTOM" | "NONE" | "PARTICIPATE" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type NotificationSettingsDialogFragment_notificationPreference$data = {
  readonly customEvents: {
    readonly failedRun: boolean;
  } | null | undefined;
  readonly global: boolean;
  readonly inherited: boolean;
  readonly namespacePath: string | null | undefined;
  readonly scope: UserNotificationPreferenceScope;
  readonly " $fragmentType": "NotificationSettingsDialogFragment_notificationPreference";
};
export type NotificationSettingsDialogFragment_notificationPreference$key = {
  readonly " $data"?: NotificationSettingsDialogFragment_notificationPreference$data;
  readonly " $fragmentSpreads": FragmentRefs<"NotificationSettingsDialogFragment_notificationPreference">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "NotificationSettingsDialogFragment_notificationPreference",
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
      "name": "global",
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

(node as any).hash = "b9a0e4b0aeabd833f553409f3e29c61f";

export default node;
