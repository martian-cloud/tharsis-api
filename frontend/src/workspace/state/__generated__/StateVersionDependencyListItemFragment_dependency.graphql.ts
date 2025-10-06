/**
 * @generated SignedSource<<407c1d3057b7050e7ca7647cc2afdb3c>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type StateVersionDependencyListItemFragment_dependency$data = {
  readonly stateVersion: {
    readonly id: string;
    readonly metadata: {
      readonly updatedAt: any;
    };
  } | null | undefined;
  readonly workspace: {
    readonly currentStateVersion: {
      readonly id: string;
    } | null | undefined;
    readonly id: string;
  } | null | undefined;
  readonly workspacePath: string;
  readonly " $fragmentType": "StateVersionDependencyListItemFragment_dependency";
};
export type StateVersionDependencyListItemFragment_dependency$key = {
  readonly " $data"?: StateVersionDependencyListItemFragment_dependency$data;
  readonly " $fragmentSpreads": FragmentRefs<"StateVersionDependencyListItemFragment_dependency">;
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
  "name": "StateVersionDependencyListItemFragment_dependency",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "workspacePath",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "StateVersion",
      "kind": "LinkedField",
      "name": "stateVersion",
      "plural": false,
      "selections": [
        (v0/*: any*/),
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
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "Workspace",
      "kind": "LinkedField",
      "name": "workspace",
      "plural": false,
      "selections": [
        (v0/*: any*/),
        {
          "alias": null,
          "args": null,
          "concreteType": "StateVersion",
          "kind": "LinkedField",
          "name": "currentStateVersion",
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
  "type": "StateVersionDependency",
  "abstractKey": null
};
})();

(node as any).hash = "bf99d8018b4d97a4b584ef755a6543a0";

export default node;
