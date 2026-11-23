/**
 * @generated SignedSource<<52c64ca271990e119c0f587516f48713>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type VCSProviderWorkspaceListItemFragment_workspace$data = {
  readonly description: string;
  readonly fullPath: string;
  readonly metadata: {
    readonly updatedAt: any;
  };
  readonly name: string;
  readonly " $fragmentType": "VCSProviderWorkspaceListItemFragment_workspace";
};
export type VCSProviderWorkspaceListItemFragment_workspace$key = {
  readonly " $data"?: VCSProviderWorkspaceListItemFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"VCSProviderWorkspaceListItemFragment_workspace">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "VCSProviderWorkspaceListItemFragment_workspace",
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

(node as any).hash = "db2ac9d6ddc1581a19f575a99c7aa9f1";

export default node;
