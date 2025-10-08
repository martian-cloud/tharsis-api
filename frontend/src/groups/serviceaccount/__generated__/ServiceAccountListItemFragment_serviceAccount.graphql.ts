/**
 * @generated SignedSource<<9a8be683cca113e7430516507ac6bcca>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ServiceAccountListItemFragment_serviceAccount$data = {
  readonly description: string;
  readonly groupPath: string;
  readonly id: string;
  readonly metadata: {
    readonly updatedAt: any;
  };
  readonly name: string;
  readonly resourcePath: string;
  readonly " $fragmentType": "ServiceAccountListItemFragment_serviceAccount";
};
export type ServiceAccountListItemFragment_serviceAccount$key = {
  readonly " $data"?: ServiceAccountListItemFragment_serviceAccount$data;
  readonly " $fragmentSpreads": FragmentRefs<"ServiceAccountListItemFragment_serviceAccount">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ServiceAccountListItemFragment_serviceAccount",
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
      "name": "resourcePath",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "groupPath",
      "storageKey": null
    }
  ],
  "type": "ServiceAccount",
  "abstractKey": null
};

(node as any).hash = "f504aa82ae91add2cdb2c2316e9b77d3";

export default node;
