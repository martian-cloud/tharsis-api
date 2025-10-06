/**
 * @generated SignedSource<<ecedef9ecbd631283a4b34a3c4a25005>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { Fragment, ReaderFragment } from 'relay-runtime';
export type JobType = "apply" | "plan" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type ManagedIdentityDetailsPolicyDetailsFragment_managedIdentity$data = {
  readonly accessRules: ReadonlyArray<{
    readonly allowedServiceAccounts: ReadonlyArray<{
      readonly id: string;
      readonly resourcePath: string;
    }>;
    readonly allowedTeams: ReadonlyArray<{
      readonly id: string;
      readonly name: string;
    }>;
    readonly allowedUsers: ReadonlyArray<{
      readonly email: string;
      readonly id: string;
      readonly username: string;
    }>;
    readonly id: string;
    readonly runStage: JobType;
  }>;
  readonly " $fragmentType": "ManagedIdentityDetailsPolicyDetailsFragment_managedIdentity";
};
export type ManagedIdentityDetailsPolicyDetailsFragment_managedIdentity$key = {
  readonly " $data"?: ManagedIdentityDetailsPolicyDetailsFragment_managedIdentity$data;
  readonly " $fragmentSpreads": FragmentRefs<"ManagedIdentityDetailsPolicyDetailsFragment_managedIdentity">;
};

const node: ReaderFragment = (function(){
var v0 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
};
return {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ManagedIdentityDetailsPolicyDetailsFragment_managedIdentity",
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "ManagedIdentityAccessRule",
      "kind": "LinkedField",
      "name": "accessRules",
      "plural": true,
      "selections": [
        (v0/*: any*/),
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
          "concreteType": "User",
          "kind": "LinkedField",
          "name": "allowedUsers",
          "plural": true,
          "selections": [
            (v0/*: any*/),
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "username",
              "storageKey": null
            },
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "email",
              "storageKey": null
            }
          ],
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "concreteType": "Team",
          "kind": "LinkedField",
          "name": "allowedTeams",
          "plural": true,
          "selections": [
            (v0/*: any*/),
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "name",
              "storageKey": null
            }
          ],
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "concreteType": "ServiceAccount",
          "kind": "LinkedField",
          "name": "allowedServiceAccounts",
          "plural": true,
          "selections": [
            (v0/*: any*/),
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
      "storageKey": null
    }
  ],
  "type": "ManagedIdentity",
  "abstractKey": null
};
})();

(node as any).hash = "c06d8dcef4050d5ac9d8fffc0a4e02ef";

export default node;
