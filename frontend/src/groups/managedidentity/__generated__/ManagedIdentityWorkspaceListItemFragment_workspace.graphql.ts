/**
 * @generated SignedSource<<c4a6993dd00881e946b72890877e2a2b>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ManagedIdentityWorkspaceListItemFragment_workspace$data = {
  readonly description: string;
  readonly fullPath: string;
  readonly id: string;
  readonly metadata: {
    readonly updatedAt: any;
  };
  readonly name: string;
  readonly " $fragmentType": "ManagedIdentityWorkspaceListItemFragment_workspace";
};
export type ManagedIdentityWorkspaceListItemFragment_workspace$key = {
  readonly " $data"?: ManagedIdentityWorkspaceListItemFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"ManagedIdentityWorkspaceListItemFragment_workspace">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ManagedIdentityWorkspaceListItemFragment_workspace",
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
      "name": "fullPath",
      "storageKey": null
    }
  ],
  "type": "Workspace",
  "abstractKey": null
};

(node as any).hash = "11d84a6fb576c1441c5b0a9648ea1d6d";

export default node;
