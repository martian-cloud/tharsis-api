/**
 * @generated SignedSource<<3c52ccd2fdb4ac793c9d6131d1826a75>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type PackageDetailsIndexFragment_package$data = {
  readonly allowMutableVersions: boolean;
  readonly description: string;
  readonly groupPath: string;
  readonly id: string;
  readonly latestVersion: {
    readonly id: string;
    readonly " $fragmentSpreads": FragmentRefs<"PackageDetailsIndexFragment_version">;
  } | null | undefined;
  readonly name: string;
  readonly " $fragmentSpreads": FragmentRefs<"PackageDetailsSidebarFragment_package" | "PackageVersionListFragment_package">;
  readonly " $fragmentType": "PackageDetailsIndexFragment_package";
};
export type PackageDetailsIndexFragment_package$key = {
  readonly " $data"?: PackageDetailsIndexFragment_package$data;
  readonly " $fragmentSpreads": FragmentRefs<"PackageDetailsIndexFragment_package">;
};

import PackageDetailsIndexRefetchQuery_graphql from './PackageDetailsIndexRefetchQuery.graphql';

const node: ReaderFragment = (function(){
var v0 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
};
return {
  "argumentDefinitions": [
    {
      "kind": "RootArgument",
      "name": "after"
    },
    {
      "kind": "RootArgument",
      "name": "before"
    },
    {
      "kind": "RootArgument",
      "name": "first"
    },
    {
      "kind": "RootArgument",
      "name": "last"
    }
  ],
  "kind": "Fragment",
  "metadata": {
    "refetch": {
      "connection": null,
      "fragmentPathInResult": [
        "node"
      ],
      "operation": PackageDetailsIndexRefetchQuery_graphql,
      "identifierInfo": {
        "identifierField": "id",
        "identifierQueryVariableName": "id"
      }
    }
  },
  "name": "PackageDetailsIndexFragment_package",
  "selections": [
    (v0/*: any*/),
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
      "name": "groupPath",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "allowMutableVersions",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "PackageVersion",
      "kind": "LinkedField",
      "name": "latestVersion",
      "plural": false,
      "selections": [
        (v0/*: any*/),
        {
          "args": null,
          "kind": "FragmentSpread",
          "name": "PackageDetailsIndexFragment_version"
        }
      ],
      "storageKey": null
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "PackageDetailsSidebarFragment_package"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "PackageVersionListFragment_package"
    }
  ],
  "type": "Package",
  "abstractKey": null
};
})();

(node as any).hash = "b57403a12bffc376f0545466fb329c64";

export default node;
