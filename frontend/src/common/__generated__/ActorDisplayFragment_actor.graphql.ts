/**
 * @generated SignedSource<<f32d1a724f5722e33ef3ea2df238fc0a>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type ActorType = "JOB" | "SCIM" | "SERVICE_ACCOUNT" | "SYSTEM" | "USER" | "VCS_PROVIDER" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type ActorDisplayFragment_actor$data = {
  readonly principalId: string | null | undefined;
  readonly trnSnapshot?: string | null | undefined;
  readonly type: ActorType;
  readonly user?: {
    readonly email: string;
  } | null | undefined;
  readonly " $fragmentType": "ActorDisplayFragment_actor";
};
export type ActorDisplayFragment_actor$key = {
  readonly " $data"?: ActorDisplayFragment_actor$data;
  readonly " $fragmentSpreads": FragmentRefs<"ActorDisplayFragment_actor">;
};

const node: ReaderFragment = (function(){
var v0 = [
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "trnSnapshot",
    "storageKey": null
  }
];
return {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ActorDisplayFragment_actor",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "type",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "principalId",
      "storageKey": null
    },
    {
      "kind": "InlineFragment",
      "selections": [
        {
          "alias": null,
          "args": null,
          "concreteType": "User",
          "kind": "LinkedField",
          "name": "user",
          "plural": false,
          "selections": [
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "email",
              "storageKey": null
            }
          ],
          "storageKey": null
        }
      ],
      "type": "UserActor",
      "abstractKey": null
    },
    {
      "kind": "InlineFragment",
      "selections": (v0/*: any*/),
      "type": "ServiceAccountActor",
      "abstractKey": null
    },
    {
      "kind": "InlineFragment",
      "selections": (v0/*: any*/),
      "type": "VCSProviderActor",
      "abstractKey": null
    },
    {
      "kind": "InlineFragment",
      "selections": (v0/*: any*/),
      "type": "JobActor",
      "abstractKey": null
    }
  ],
  "type": "Actor",
  "abstractKey": "__isActor"
};
})();

(node as any).hash = "ad7a6b7367e66269655b5ce95c369f0a";

export default node;
