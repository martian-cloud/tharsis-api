/**
 * @generated SignedSource<<9ddff59df238a53e8b826cdccb48aed1>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type StateVersionDependenciesFragment_dependencies$data = {
  readonly dependencies: ReadonlyArray<{
    readonly workspacePath: string;
    readonly " $fragmentSpreads": FragmentRefs<"StateVersionDependencyListItemFragment_dependency">;
  }>;
  readonly " $fragmentType": "StateVersionDependenciesFragment_dependencies";
};
export type StateVersionDependenciesFragment_dependencies$key = {
  readonly " $data"?: StateVersionDependenciesFragment_dependencies$data;
  readonly " $fragmentSpreads": FragmentRefs<"StateVersionDependenciesFragment_dependencies">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "StateVersionDependenciesFragment_dependencies",
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "StateVersionDependency",
      "kind": "LinkedField",
      "name": "dependencies",
      "plural": true,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "workspacePath",
          "storageKey": null
        },
        {
          "args": null,
          "kind": "FragmentSpread",
          "name": "StateVersionDependencyListItemFragment_dependency"
        }
      ],
      "storageKey": null
    }
  ],
  "type": "StateVersion",
  "abstractKey": null
};

(node as any).hash = "bfcee1bde42cb4e0fb3899e9254b0e0c";

export default node;
