/**
 * @generated SignedSource<<b2006fa0755d280578269d32cb46da76>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type UserNotificationPreferenceScope = "ALL" | "CUSTOM" | "NONE" | "PARTICIPATE" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type NotificationButtonFragment_notificationPreference$data = {
  readonly customEvents: {
    readonly failedRun: boolean;
    readonly serviceAccountSecretExpiration: boolean;
  } | null | undefined;
  readonly global: boolean;
  readonly inherited: boolean;
  readonly namespacePath: string | null | undefined;
  readonly scope: UserNotificationPreferenceScope;
  readonly " $fragmentType": "NotificationButtonFragment_notificationPreference";
};
export type NotificationButtonFragment_notificationPreference$key = {
  readonly " $data"?: NotificationButtonFragment_notificationPreference$data;
  readonly " $fragmentSpreads": FragmentRefs<"NotificationButtonFragment_notificationPreference">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "NotificationButtonFragment_notificationPreference",
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
      "kind": "ScalarField",
      "name": "global",
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
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "serviceAccountSecretExpiration",
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "UserNotificationPreference",
  "abstractKey": null
};

(node as any).hash = "5a96a8ff145cb528438f42310f18c375";

export default node;
