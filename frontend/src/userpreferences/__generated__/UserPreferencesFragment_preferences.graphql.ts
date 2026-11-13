/**
 * @generated SignedSource<<debd3bdedd8a2e970df08be480c0bb50>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type UserPreferencesFragment_preferences$data = {
  readonly me: {
    readonly user: {
      readonly " $fragmentSpreads": FragmentRefs<"UserSessionsFragment_user">;
    } | null | undefined;
  } | null | undefined;
  readonly userPreferences: {
    readonly globalPreferences: {
      readonly " $fragmentSpreads": FragmentRefs<"GlobalNotificationPreferenceFragment_notificationPreference">;
    };
  };
  readonly " $fragmentType": "UserPreferencesFragment_preferences";
};
export type UserPreferencesFragment_preferences$key = {
  readonly " $data"?: UserPreferencesFragment_preferences$data;
  readonly " $fragmentSpreads": FragmentRefs<"UserPreferencesFragment_preferences">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "UserPreferencesFragment_preferences",
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "UserPreferences",
      "kind": "LinkedField",
      "name": "userPreferences",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "concreteType": "GlobalUserPreferences",
          "kind": "LinkedField",
          "name": "globalPreferences",
          "plural": false,
          "selections": [
            {
              "args": null,
              "kind": "FragmentSpread",
              "name": "GlobalNotificationPreferenceFragment_notificationPreference"
            }
          ],
          "storageKey": null
        }
      ],
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": null,
      "kind": "LinkedField",
      "name": "me",
      "plural": false,
      "selections": [
        {
          "fragment": {
            "kind": "InlineFragment",
            "selections": [
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "UserSessionsFragment_user"
              }
            ],
            "type": "User",
            "abstractKey": null
          },
          "kind": "AliasedInlineFragmentSpread",
          "name": "user"
        }
      ],
      "storageKey": null
    }
  ],
  "type": "Query",
  "abstractKey": null
};

(node as any).hash = "4a69744ed172b4ceeda22294d4d0c471";

export default node;
