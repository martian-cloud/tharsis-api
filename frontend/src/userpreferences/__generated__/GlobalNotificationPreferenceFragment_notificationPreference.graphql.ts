/**
 * @generated SignedSource<<18b002f86cd64fd1ae308fe4a4a65014>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GlobalNotificationPreferenceFragment_notificationPreference$data = {
  readonly notificationPreference: {
    readonly " $fragmentSpreads": FragmentRefs<"NotificationButtonFragment_notificationPreference">;
  };
  readonly " $fragmentType": "GlobalNotificationPreferenceFragment_notificationPreference";
};
export type GlobalNotificationPreferenceFragment_notificationPreference$key = {
  readonly " $data"?: GlobalNotificationPreferenceFragment_notificationPreference$data;
  readonly " $fragmentSpreads": FragmentRefs<"GlobalNotificationPreferenceFragment_notificationPreference">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "GlobalNotificationPreferenceFragment_notificationPreference",
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "UserNotificationPreference",
      "kind": "LinkedField",
      "name": "notificationPreference",
      "plural": false,
      "selections": [
        {
          "args": null,
          "kind": "FragmentSpread",
          "name": "NotificationButtonFragment_notificationPreference"
        }
      ],
      "storageKey": null
    }
  ],
  "type": "GlobalUserPreferences",
  "abstractKey": null
};

(node as any).hash = "55466bc6c03fe30577704fc0aaf30f81";

export default node;
