/**
 * @generated SignedSource<<73cdcd3936111c5654c4b36fcfa01575>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type AssignedServiceAccountListItemFragment_assignedServiceAccount$data = {
  readonly description: string;
  readonly groupPath: string;
  readonly id: string;
  readonly metadata: {
    readonly updatedAt: any;
  };
  readonly name: string;
  readonly resourcePath: string;
  readonly " $fragmentType": "AssignedServiceAccountListItemFragment_assignedServiceAccount";
};
export type AssignedServiceAccountListItemFragment_assignedServiceAccount$key = {
  readonly " $data"?: AssignedServiceAccountListItemFragment_assignedServiceAccount$data;
  readonly " $fragmentSpreads": FragmentRefs<"AssignedServiceAccountListItemFragment_assignedServiceAccount">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "AssignedServiceAccountListItemFragment_assignedServiceAccount",
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
      "name": "name",
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
  "type": "ServiceAccount",
  "abstractKey": null
};

(node as any).hash = "e81bffa171d44af222ceb03d73c289f0";

export default node;
