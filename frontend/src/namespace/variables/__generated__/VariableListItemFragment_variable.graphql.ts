/**
 * @generated SignedSource<<727bfb519772a4c57a23a3f81186e006>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type VariableCategory = "environment" | "terraform" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type VariableListItemFragment_variable$data = {
  readonly category: VariableCategory;
  readonly id: string;
  readonly key: string;
  readonly latestVersionId: string;
  readonly metadata: {
    readonly updatedAt: any;
  };
  readonly namespacePath: string;
  readonly sensitive: boolean;
  readonly value: string | null | undefined;
  readonly " $fragmentType": "VariableListItemFragment_variable";
};
export type VariableListItemFragment_variable$key = {
  readonly " $data"?: VariableListItemFragment_variable$data;
  readonly " $fragmentSpreads": FragmentRefs<"VariableListItemFragment_variable">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "VariableListItemFragment_variable",
  "selections": [
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
      "name": "key",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "category",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "sensitive",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "value",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "namespacePath",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "latestVersionId",
      "storageKey": null
    },
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
    }
  ],
  "type": "NamespaceVariable",
  "abstractKey": null
};

(node as any).hash = "c2cea86d899ff72000865bbcb9e175a4";

export default node;
