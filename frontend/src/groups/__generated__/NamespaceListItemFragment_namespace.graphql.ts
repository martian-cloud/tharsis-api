/**
 * @generated SignedSource<<dbf1604291a8361b71bf9b26b905ea3f>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type NamespaceListItemFragment_namespace$data = {
  readonly __typename: string;
  readonly description: string;
  readonly destroyed?: boolean;
  readonly fullPath: string;
  readonly hasChildren?: boolean;
  readonly id: string;
  readonly labels?: ReadonlyArray<{
    readonly key: string;
    readonly value: string;
  }>;
  readonly name: string;
  readonly " $fragmentType": "NamespaceListItemFragment_namespace";
};
export type NamespaceListItemFragment_namespace$key = {
  readonly " $data"?: NamespaceListItemFragment_namespace$data;
  readonly " $fragmentSpreads": FragmentRefs<"NamespaceListItemFragment_namespace">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "NamespaceListItemFragment_namespace",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "__typename",
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
    },
    {
      "kind": "InlineFragment",
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "hasChildren",
          "storageKey": null
        }
      ],
      "type": "Group",
      "abstractKey": null
    },
    {
      "kind": "InlineFragment",
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "destroyed",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "concreteType": "WorkspaceLabel",
          "kind": "LinkedField",
          "name": "labels",
          "plural": true,
          "selections": [
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "key",
              "storageKey": null
            },
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "value",
              "storageKey": null
            }
          ],
          "storageKey": null
        }
      ],
      "type": "Workspace",
      "abstractKey": null
    }
  ],
  "type": "Namespace",
  "abstractKey": "__isNamespace"
};

(node as any).hash = "7e26e6a1426863e6458ccd0eabb7baf4";

export default node;
