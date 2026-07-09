/**
 * @generated SignedSource<<b805d4e11b411bac1f593a467b84ddc7>>
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
  "type": "StateVersionInventory",
  "abstractKey": null
};

(node as any).hash = "42320b4d2c2c6af1ebd1589efe0557ed";

export default node;
