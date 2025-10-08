/**
 * @generated SignedSource<<e5d68563ff9af6ea7b2a2b338ad5f4c3>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ManagedIdentityAliasesListItemFragment_managedIdentity$data = {
  readonly description: string;
  readonly groupPath: string;
  readonly id: string;
  readonly metadata: {
    readonly updatedAt: any;
  };
  readonly name: string;
  readonly type: string;
  readonly " $fragmentType": "ManagedIdentityAliasesListItemFragment_managedIdentity";
};
export type ManagedIdentityAliasesListItemFragment_managedIdentity$key = {
  readonly " $data"?: ManagedIdentityAliasesListItemFragment_managedIdentity$data;
  readonly " $fragmentSpreads": FragmentRefs<"ManagedIdentityAliasesListItemFragment_managedIdentity">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ManagedIdentityAliasesListItemFragment_managedIdentity",
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "ResourceMetadata",
      "kind": "LinkedField",
      "name": "metadata",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "updatedAt",
          "storageKey": null
        }
      ],
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "id",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "name",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "description",
      "storageKey": null
    },
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
      "name": "groupPath",
      "storageKey": null
    }
  ],
  "type": "ManagedIdentity",
  "abstractKey": null
};

(node as any).hash = "02b15fceedea89fa8bd5088ae701fce0";

export default node;
