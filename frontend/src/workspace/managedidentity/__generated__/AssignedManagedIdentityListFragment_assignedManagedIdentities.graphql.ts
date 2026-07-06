/**
 * @generated SignedSource<<ba71d2e9bd3f2df8519aa723d5fc7ee8>>
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
    readonly edges: ReadonlyArray<{
      readonly node: {
        readonly id: string;
      } | null | undefined;
    } | null | undefined> | null | undefined;
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

(node as any).hash = "f2c2779cfb52c5583939cd2284ecde86";

export default node;
