/**
 * @generated SignedSource<<4cccc8062112b9ea8dc7282d30164807>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type AssignedManagedIdentityListFragment_assignedManagedIdentities$data = {
  readonly assignedManagedIdentities: ReadonlyArray<{
    readonly id: string;
    readonly " $fragmentSpreads": FragmentRefs<"AssignedManagedIdentityListItemFragment_managedIdentity">;
  }>;
  readonly fullPath: string;
  readonly id: string;
  readonly managedIdentities: {
    readonly totalCount: number;
  };
  readonly " $fragmentType": "AssignedManagedIdentityListFragment_assignedManagedIdentities";
};
export type AssignedManagedIdentityListFragment_assignedManagedIdentities$key = {
  readonly " $data"?: AssignedManagedIdentityListFragment_assignedManagedIdentities$data;
  readonly " $fragmentSpreads": FragmentRefs<"AssignedManagedIdentityListFragment_assignedManagedIdentities">;
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
  "name": "AssignedManagedIdentityListFragment_assignedManagedIdentities",
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
      "args": [
        {
          "kind": "Literal",
          "name": "first",
          "value": 0
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
          "kind": "ScalarField",
          "name": "totalCount",
          "storageKey": null
        }
      ],
      "storageKey": "managedIdentities(first:0,includeInherited:true)"
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

(node as any).hash = "021e0f64a4633b2a3e39acb96e63c9ff";

export default node;
