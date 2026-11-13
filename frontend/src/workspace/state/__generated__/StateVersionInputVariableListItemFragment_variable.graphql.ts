/**
 * @generated SignedSource<<046073946f09c382e12635b371cc8ce5>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type VariableCategory = "environment" | "terraform" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type StateVersionInputVariableListItemFragment_variable$data = {
  readonly category: VariableCategory;
  readonly includedInTfConfig: boolean | null | undefined;
  readonly key: string;
  readonly namespacePath: string | null | undefined;
  readonly sensitive: boolean;
  readonly value: string | null | undefined;
  readonly versionId: string | null | undefined;
  readonly " $fragmentType": "StateVersionInputVariableListItemFragment_variable";
};
export type StateVersionInputVariableListItemFragment_variable$key = {
  readonly " $data"?: StateVersionInputVariableListItemFragment_variable$data;
  readonly " $fragmentSpreads": FragmentRefs<"StateVersionInputVariableListItemFragment_variable">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "StateVersionInputVariableListItemFragment_variable",
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

(node as any).hash = "347702717bce6e3e810aa6c9ee448db7";

export default node;
