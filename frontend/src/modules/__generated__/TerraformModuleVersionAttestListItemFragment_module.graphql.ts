/**
 * @generated SignedSource<<183c6d1235bad1c5dbe1e9c08fe90453>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type TerraformModuleVersionAttestListItemFragment_module$data = {
  readonly createdByActor: {
    readonly " $fragmentSpreads": FragmentRefs<"ActorDisplayFragment_actor">;
  } | null | undefined;
  readonly data: string;
  readonly description: string;
  readonly id: string;
  readonly metadata: {
    readonly createdAt: any;
  };
  readonly predicateType: string;
  readonly " $fragmentType": "TerraformModuleVersionAttestListItemFragment_module";
};
export type TerraformModuleVersionAttestListItemFragment_module$key = {
  readonly " $data"?: TerraformModuleVersionAttestListItemFragment_module$data;
  readonly " $fragmentSpreads": FragmentRefs<"TerraformModuleVersionAttestListItemFragment_module">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "TerraformModuleVersionAttestListItemFragment_module",
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
      "name": "description",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "predicateType",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "data",
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
          "name": "createdAt",
          "storageKey": null
        }
      ],
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": null,
      "kind": "LinkedField",
      "name": "createdByActor",
      "plural": false,
      "selections": [
        {
          "args": null,
          "kind": "FragmentSpread",
          "name": "ActorDisplayFragment_actor"
        }
      ],
      "storageKey": null
    }
  ],
  "type": "TerraformModuleAttestation",
  "abstractKey": null
};

(node as any).hash = "26f5f71d5aed31e73c0fe2f2b5d7ce28";

export default node;
