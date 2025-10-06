/**
 * @generated SignedSource<<e5f0825cd39c1bbffcdd32714dced377>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type VariableCategory = "environment" | "terraform" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type RunVariableListItemFragment_variable$data = {
  readonly category: VariableCategory;
  readonly includedInTfConfig: boolean | null | undefined;
  readonly key: string;
  readonly namespacePath: string | null | undefined;
  readonly sensitive: boolean;
  readonly value: string | null | undefined;
  readonly versionId: string | null | undefined;
  readonly " $fragmentType": "RunVariableListItemFragment_variable";
};
export type RunVariableListItemFragment_variable$key = {
  readonly " $data"?: RunVariableListItemFragment_variable$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunVariableListItemFragment_variable">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "RunVariableListItemFragment_variable",
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
      "name": "category",
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
      "name": "sensitive",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "versionId",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "includedInTfConfig",
      "storageKey": null
    }
  ],
  "type": "RunVariable",
  "abstractKey": null
};

(node as any).hash = "3c0c0d362359e22829912b6b5ab20a0a";

export default node;
