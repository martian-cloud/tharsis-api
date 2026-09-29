/**
 * @generated SignedSource<<7f073ca672b6f8abc10470246be6c9d9>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type AssignedManagedIdentityListAssignedFragment_assignedManagedIdentities$data = {
  readonly assignedManagedIdentities: ReadonlyArray<{
    readonly id: string;
    readonly " $fragmentSpreads": FragmentRefs<"AssignedManagedIdentityListItemFragment_managedIdentity">;
  }>;
  readonly managedIdentities: {
    readonly edges: ReadonlyArray<{
      readonly node: {
        readonly id: string;
      } | null | undefined;
    } | null | undefined> | null | undefined;
  };
  readonly " $fragmentType": "AssignedManagedIdentityListAssignedFragment_assignedManagedIdentities";
};
export type AssignedManagedIdentityListAssignedFragment_assignedManagedIdentities$key = {
  readonly " $data"?: AssignedManagedIdentityListAssignedFragment_assignedManagedIdentities$data;
  readonly " $fragmentSpreads": FragmentRefs<"AssignedManagedIdentityListAssignedFragment_assignedManagedIdentities">;
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
  "name": "AssignedManagedIdentityListAssignedFragment_assignedManagedIdentities",
  "selections": [
    {
      "alias": null,
      "args": [
        {
          "kind": "Literal",
          "name": "first",
          "value": 1
        },
        {
          "kind": "Literal",
          "name": "includeInherited",
          "value": true
        }
      ],
      "concreteType": "ManagedIdentityConnection",
      "kind": "LinkedField",
      "name": "managedIdentities",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "concreteType": "ManagedIdentityEdge",
          "kind": "LinkedField",
          "name": "edges",
          "plural": true,
          "selections": [
            {
              "alias": null,
              "args": null,
              "concreteType": "ManagedIdentity",
              "kind": "LinkedField",
              "name": "node",
              "plural": false,
              "selections": [
                (v0/*: any*/)
              ],
              "storageKey": null
            }
          ],
          "storageKey": null
        }
      ],
      "storageKey": "managedIdentities(first:1,includeInherited:true)"
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "ManagedIdentity",
      "kind": "LinkedField",
      "name": "assignedManagedIdentities",
      "plural": true,
      "selections": [
        (v0/*: any*/),
        {
          "args": null,
          "kind": "FragmentSpread",
          "name": "AssignedManagedIdentityListItemFragment_managedIdentity"
        }
      ],
      "storageKey": null
    }
  ],
  "type": "Workspace",
  "abstractKey": null
};
})();

(node as any).hash = "02c022e133115a09ca2b7e24a1206c4d";

export default node;
